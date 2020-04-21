package main

import "testing"

func TestCase1(t *testing.T) {
    in := `GET / HTTP/1.1
HOST: baidu.com`
    req, err := parse_request(in)
    check(err)
    status_code := write_response(req)
    if status_code != 200 {
        t.Errorf("case1: resp = %d; expected 200", status_code)
    }
}

func TestCase2(t *testing.T) {
    in := `POST / HTTP/1.1
Host: foo.com
Content-Type: application/x-www-form-urlencoded
Content-Length: 13

say=Hi&to=Mom`
    req, err := parse_request(in)
    check(err)
    status_code := write_response(req)
    if status_code != 200 {
        t.Errorf("case2: resp = %d; expected 200", status_code)
    }
}

func TestCase3(t *testing.T) {
    in := `POST /test.html HTTP/1.1
Host: example.org
Content-Type: multipart/form-data;boundary=boundary

--boundary
Content-Disposition: form-data; name="field1"

value1

--boundary
Content-Disposition: form-data; name="field2"; filename="example.txt"

value2

--boundary`
    req, err := parse_request(in)
    check(err)
    status_code := write_response(req)
    if status_code != 200 {
        t.Errorf("case3: resp = %d; expected 200", status_code)
    }
}
