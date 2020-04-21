package  main

import (
    "strings"
    "flag"
    "errors"
    "net/url"
    "io/ioutil"
    "fmt"
)


type Header struct{
    Name string
    Value string
}

type File struct {
    FileName string
    Content string
}

type PostData struct {
    Name string
    Content string
}

type Request struct{
    Method string
    Path string
    Version string
    Host string
    Headers []Header
    Body []byte
    Files []File
    Datas []PostData
}

type Response struct{
    Version string
    Code uint
    Headers []Header
    Body []byte
}


func check(e error) {
    if e != nil {
        panic(e)
    }
}

func (req *Request) add_header(header Header) []Header {
    req.Headers = append(req.Headers, header)
    return req.Headers
}

func (req *Request) add_file(file File) []File {
    req.Files = append(req.Files, file)
    return req.Files
}

func (req *Request) add_data(post_data PostData) []PostData {
    req.Datas = append(req.Datas, post_data)
    return req.Datas
}

func parse_request(text string) (*Request, error) {
    http_lines := strings.Split(text, "\n")
    request_info := strings.Split(http_lines[0], " ")
    method := request_info[0]
    path := request_info[1]
    protocol_info := request_info[2]
    version := strings.Split(protocol_info, "/")[1]
    host_info := strings.Split(http_lines[1], ":")
    host := host_info[1]

    req := Request{Method: method, Path:path, Version: version, Host: host}

    content_type := ""
    content_bounty := ""
    body_index := 0
    for i, content := range http_lines[2:] {

        if content == "" {
            body_index = i
            break
        }
        header_info := strings.Split(content, ":")
        name := strings.TrimSpace(header_info[0])
        value := strings.TrimSpace(header_info[1])
        if name == "Content-Type" {
            if strings.HasPrefix(value, "multipart/form-data") {
                content_type_info := strings.Split(value, ";")
                value = content_type_info[0]
                content_bounty_info := content_type_info[1]
                content_bounty = strings.Split(content_bounty_info, "=")[1]
            }
            content_type = value
        }
        req.add_header(Header{Name:name, Value:value})
    }

    is_real_content := false
    field_name := ""
    file_name := ""
    field_content := ""
    if len(http_lines) < 3 {
        return &req, nil
    }
    for _, content := range http_lines[body_index+3:] {
        if content_type == "" {
            return &req, errors.New("can't find content type")
        }
        req.Body = []byte(content)
        if content_type == "application/x-www-form-urlencoded" {
            params, err := url.ParseQuery(content)
            for k, v := range params {
                req.add_data(PostData{Name: k, Content:v[0]})
            }
            check(err)
        } else if content_type == "multipart/form-data" {
            if content == "--" + content_bounty {
                if is_real_content {
                    strings.TrimSuffix(field_content, "\n")
                    if file_name != "" {
                        req.add_file(File{FileName: file_name, Content:field_content})
                    } else {
                        req.add_data(PostData{Name: field_name, Content:field_content})
                    }
                    is_real_content = false
                    field_name = ""
                    file_name = ""
                    field_content = ""
                }
                continue
            }
            if content == "" {
                is_real_content = true
                continue
            }
            if strings.HasPrefix(content, "Content-Disposition") {
                body_header_info := strings.Split(content, ";")
                if strings.TrimSpace(strings.Split(body_header_info[0], ":")[1]) != "form-data" {
                    return &req, errors.New("Unsupported Content-Disposition type")
                }
                for _, c := range body_header_info[1:] {
                    left_body_header_info := strings.Split(c, "=")
                    left_body_header_key := strings.TrimSpace(left_body_header_info[0])
                    if left_body_header_key == "name" {
                        field_name = strings.Trim(left_body_header_info[1], "\"")
                    } else if left_body_header_key == "filename" {
                        file_name = strings.Trim(left_body_header_info[1], "\"")
                    } else {
                        return &req, errors.New("Unsupported body header")
                    }
                }
            } else if strings.HasPrefix(content, "Content-Type") {
                return &req, errors.New("Unsupported Body Content Type")
            }
            if is_real_content {
                field_content += content
            }
        } else {
            return &req, errors.New("Unsupported content")
        }
    }
    return &req, nil
}


func write_response(req *Request) uint {
    resp := Response{Version: "1.1", Code: 200}
    fmt.Println(resp)
    return resp.Code
}


func main() {
    fn := flag.String("fn", "", "A file containing http structure information")
    flag.Parse()
    content, err := ioutil.ReadFile(*fn)
    check(err)
    text := string(content)
    req, err := parse_request(text)
    check(err)
    write_response(req)
}
