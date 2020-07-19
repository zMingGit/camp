package main

import (
    "bufio"
    "strings"
    // utils "./utils"
    "fmt"
)

// A Header represents the key-value pairs in an HTTP header.
type Header map[string][]string

type Request struct{
    Method string
    URL string
    RequestURI string
    Proto string
    ProtoMajor int
    ProtoMinor int
    ContentLength int64
    Headers  map[string][] string
    TransferEncoding []string
}

type badStringError struct {
    what string
    str  string
}

func (e *badStringError) Error() string { return fmt.Sprintf("%s %q", e.what, e.str) }

func chunked(te []string) bool { return len(te) > 0 && te[0] == "chunked" }

func fixTransferEncoding(req *Request) error {
	raw, present := req.Headers["Transfer-Encoding"]
	if !present {
		return nil
	}
	delete(req.Headers, "Transfer-Encoding")

	encodings := strings.Split(raw[0], ",")
	te := make([]string, 0, len(encodings))
	// TODO: Even though we only support "identity" and "chunked"
	// encodings, the loop below is designed with foresight. One
	// invariant that must be maintained is that, if present,
	// chunked encoding must always come first.
	for _, encoding := range encodings {
		encoding = strings.ToLower(strings.TrimSpace(encoding))
		// "identity" encoding is not recorded
		if encoding == "identity" {
			break
		}
		if encoding != "chunked" {
			return &badStringError{"unsupported transfer encoding", encoding}
		}
		te = te[0 : len(te)+1]
		te[len(te)-1] = encoding
	}
	if len(te) > 1 {
		return &badStringError{"too many transfer encodings", strings.Join(te, ",")}
	}
	if len(te) > 0 {
		// RFC 7230 3.3.2 says "A sender MUST NOT send a
		// Content-Length header field in any message that
		// contains a Transfer-Encoding header field."
		//
		// but also:
		// "If a message is received with both a
		// Transfer-Encoding and a Content-Length header
		// field, the Transfer-Encoding overrides the
		// Content-Length. Such a message might indicate an
		// attempt to perform request smuggling (Section 9.5)
		// or response splitting (Section 9.4) and ought to be
		// handled as an error. A sender MUST remove the
		// received Content-Length field prior to forwarding
		// such a message downstream."
		//
		// Reportedly, these appear in the wild.
		delete(req.Headers, "Content-Length")
		req.TransferEncoding = te
		return nil
	}

	return nil
}

// func fixLength(status int, requestMethod string, header Header, te []string) (int64, error) {
// 	contentLens := header["Content-Length"]
//
// 	// Hardening against HTTP request smuggling
// 	if len(contentLens) > 1 {
// 		// Per RFC 7230 Section 3.3.2, prevent multiple
// 		// Content-Length headers if they differ in value.
// 		// If there are dups of the value, remove the dups.
// 		// See Issue 16490.
// 		first := strings.TrimSpace(contentLens[0])
// 		for _, ct := range contentLens[1:] {
// 			if first != strings.TrimSpace(ct) {
// 				return 0, fmt.Errorf("http: message cannot contain multiple Content-Length headers; got %q", contentLens)
// 			}
// 		}
//
// 		// deduplicate Content-Length
// 		header.Del("Content-Length")
// 		header.Add("Content-Length", first)
//
// 		contentLens = header["Content-Length"]
// 	}
//
// 	// Logic based on response type or status
// 	if noResponseBodyExpected(requestMethod) {
// 		// For HTTP requests, as part of hardening against request
// 		// smuggling (RFC 7230), don't allow a Content-Length header for
// 		// methods which don't permit bodies. As an exception, allow
// 		// exactly one Content-Length header if its value is "0".
// 		if len(contentLens) > 0 && !(len(contentLens) == 1 && contentLens[0] == "0") {
// 			return 0, fmt.Errorf("http: method cannot contain a Content-Length; got %q", contentLens)
// 		}
// 		return 0, nil
// 	}
// 	if status/100 == 1 {
// 		return 0, nil
// 	}
// 	switch status {
// 	case 204, 304:
// 		return 0, nil
// 	}
//
// 	// Logic based on Transfer-Encoding
// 	if chunked(te) {
// 		return -1, nil
// 	}
//
// 	// Logic based on Content-Length
// 	var cl string
// 	if len(contentLens) == 1 {
// 		cl = strings.TrimSpace(contentLens[0])
// 	}
// 	if cl != "" {
// 		n, err := parseContentLength(cl)
// 		if err != nil {
// 			return -1, err
// 		}
// 		return n, nil
// 	}
//     delete(header, "Content-Length")
//
// 	return 0, nil
// }

func readLine(b *bufio.Reader) (string, error) {
    var line []byte
    for {
        l, more, err := b.ReadLine()
        if err != nil {
            return "", err
        }
        if line == nil && !more {
	    	return string(l), nil
	    }
        line = append(line, l...)
        if !more {
            break
        }
    }
    return string(line), nil
}

func readLineSlice(r *bufio.Reader) ([]byte, error) {
    var line []byte
    for {
        l, more, err := r.ReadLine()
        if err != nil {
            return nil, err
        }
        // Avoid the copy if the first call produced a full line.
        if line == nil && !more {
            return l, nil
        }
        line = append(line, l...)
        if !more {
            break
        }
    }
    return line, nil
}

func skipSpace(r *bufio.Reader) int {
    n := 0
    for {
        c, err := r.ReadByte()
        if err != nil {
            // Bufio will keep err until next read.
            break
        }
        if c != ' ' && c != '\t' {
            r.UnreadByte()
            break
        }
        n++
    }
    return n
}

func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
    s1 := strings.Index(line, " ")
    s2 := strings.Index(line[s1+1:], " ")
    if s1 < 0 || s2 < 0 {
        return
    }
    s2 += s1 + 1
    return line[:s1], line[s1+1 : s2], line[s2+1:], true
}

func readRequest(b *bufio.Reader) (req *Request, err error){
    tp := NewReader(b)
    var s string
    if s, err = tp.ReadLine(); err != nil {
        return nil, err
    }
    req = new(Request)
    var ok bool
    req.Method, req.RequestURI, req.Proto, ok = parseRequestLine(s)
    if !ok {
        return nil, &badStringError{"malformed HTTP request", s}
    }
    // req.Headers, err = tp.ReadMIMEHeader()
    // if err != nil {
    //     return nil, err
    // }
    // fixTransferEncoding(req)
    // is_chunked := chunked()
    // fmt.Println(req.Headers)
    for {
        s, err = tp.ReadLine()
        if err != nil {
            fmt.Println("fuck")
        }
        fmt.Println(s)
    }
    // utils.Check(err)
    return nil, err
}
