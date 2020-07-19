package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "path"
    "os"
    "io"
    utils "./utils"
    "fmt"
    "net"
    "net/http"
    "io/ioutil"
)


type Configuration struct {
    FileDir    string
}


type Words struct {
    current_line []byte
	prev_c byte
}


const defaultBufSize = 4096

func newResponse(body string) (*http.Response) {
    return &http.Response{
      Status:        "200 OK",
      StatusCode:    200,
      Proto:         "HTTP/1.1",
      ProtoMajor:    1,
      ProtoMinor:    1,
      Body:          ioutil.NopCloser(bytes.NewBufferString(body)),
      ContentLength: int64(len(body)),
      Header:        make(http.Header, 0),
    }
}


func init_config() Configuration {
    file, _ := os.Open("./config.json")
    defer file.Close()
    configuration := Configuration{}

    decoder := json.NewDecoder(file)
    err := decoder.Decode(&configuration)
    utils.Check(err)
    return configuration
}


func handleConnection(conn net.Conn, conf Configuration) {
    defer conn.Close()
    r := bufio.NewReader(conn)
    for {
        req, err := http.ReadRequest(r)
        if err == io.EOF {
            fmt.Println("连接被关闭")
            return
        }
        switch req.Method {
            case "GET":
                if req.URL != nil {
                    fn := path.Join(conf.FileDir, req.URL.Path)
                    content, err := ioutil.ReadFile(fn)
                    utils.Check(err)
                    resp := newResponse(string(content))
                    err = resp.Write(conn)
                    utils.Check(err)
                    fmt.Println("请求读取文件", req.URL.Path)
                }
            case "POST":
                req.ParseMultipartForm(5 * 1024 * 1024)
                if req.URL != nil {
                    f, err := req.MultipartForm.File["file"][0].Open()
                    utils.Check(err)
                    out, err := os.Create(path.Join(conf.FileDir, req.URL.Path))
                    utils.Check(err)
                    io.Copy(out, f)
                    resp := newResponse("")
                    err = resp.Write(conn)
                    utils.Check(err)
                    fmt.Println("成功写入文件", req.URL.Path)
                }
                //
                // utils.Check(err)
                // os.Copy(f)
        }
        break
        // utils.Check(err)
    }

    // scanner := bufio.NewScanner(conn)
    //for scanner.Scan() {
    //    handle_input(scanner.Text())
    //}
}

func start_server(conf Configuration) {
    ln, err := net.Listen("tcp4", ":18080")
    utils.Check(err)
    fmt.Println("init server")
    for {
        conn, err := ln.Accept()
        utils.Check(err)
        go handleConnection(conn, conf)
    }
}

func main() {
    configuration := init_config()
    fmt.Println(configuration)
    start_server(configuration)
}
