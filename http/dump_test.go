package http

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"testing"
)

type dumpTest struct {
	Req  http.Request
	Body interface{} // optional []byte or func() io.ReadCloser to populate Req.Body

	WantHeader string
	WantBody   string
	NoBody     bool // if true, set DumpRequest{,Out} body to false
}

var dumpTests = []dumpTest{

	// HTTP/1.1 => chunked coding; body; empty trailer
	{
		Req: http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   "www.google.com",
				Path:   "/search",
			},
			ProtoMajor:       1,
			ProtoMinor:       1,
			TransferEncoding: []string{"chunked"},
		},

		Body: []byte("abcdef"),

		WantHeader: "GET /search HTTP/1.1\r\n" +
			"Host: www.google.com\r\n" +
			"Transfer-Encoding: chunked\r\n",
		WantBody: "abcdef",
	},

	// Verify that DumpRequest preserves the HTTP version number, doesn't add a Host,
	// and doesn't add a User-Agent.
	{
		Req: http.Request{
			Method:     "GET",
			URL:        mustParseURL("/foo"),
			ProtoMajor: 1,
			ProtoMinor: 0,
			Header: http.Header{
				"X-Foo": []string{"X-Bar"},
			},
		},

		WantHeader: "GET /foo HTTP/1.0\r\n" +
			"X-Foo: X-Bar\r\n",
	},

	// Request with Body > 8196 (default buffer size)
	{
		Req: http.Request{
			Method: "POST",
			URL: &url.URL{
				Scheme: "http",
				Host:   "post.tld",
				Path:   "/",
			},
			Header: http.Header{
				"Content-Length": []string{"8193"},
			},

			ContentLength: 8193,
			ProtoMajor:    1,
			ProtoMinor:    1,
		},

		Body: bytes.Repeat([]byte("a"), 8193),

		WantHeader: "POST / HTTP/1.1\r\n" +
			"Host: post.tld\r\n" +
			"Content-Length: 8193\r\n",
		WantBody: strings.Repeat("a", 8193),
	},

	{
		Req: *mustReadRequest("GET http://foo.com/ HTTP/1.1\r\n" +
			"User-Agent: blah\r\n\r\n"),
		NoBody: true,
		WantHeader: "GET http://foo.com/ HTTP/1.1\r\n" +
			"User-Agent: blah\r\n",
	},

	// Issue #7215. DumpRequest should return the "Content-Length" when set
	{
		Req: *mustReadRequest("POST /v2/api/?login HTTP/1.1\r\n" +
			"Host: passport.myhost.com\r\n" +
			"Content-Length: 3\r\n" +
			"\r\nkey1=name1&key2=name2"),
		WantHeader: "POST /v2/api/?login HTTP/1.1\r\n" +
			"Host: passport.myhost.com\r\n" +
			"Content-Length: 3\r\n",
		WantBody: "key",
	},

	// Issue #7215. DumpRequest should return the "Content-Length" in ReadRequest
	{
		Req: *mustReadRequest("POST /v2/api/?login HTTP/1.1\r\n" +
			"Host: passport.myhost.com\r\n" +
			"Content-Length: 0\r\n" +
			"\r\nkey1=name1&key2=name2"),
		WantHeader: "POST /v2/api/?login HTTP/1.1\r\n" +
			"Host: passport.myhost.com\r\n" +
			"Content-Length: 0\r\n",
	},

	// Issue #7215. DumpRequest should not return the "Content-Length" if unset
	{
		Req: *mustReadRequest("POST /v2/api/?login HTTP/1.1\r\n" +
			"Host: passport.myhost.com\r\n" +
			"\r\nkey1=name1&key2=name2"),
		WantHeader: "POST /v2/api/?login HTTP/1.1\r\n" +
			"Host: passport.myhost.com\r\n",
	},
}

func TestDumpRequest(t *testing.T) {
	numg0 := runtime.NumGoroutine()
	for i, tt := range dumpTests {
		setBody := func() {
			if tt.Body == nil {
				return
			}
			switch b := tt.Body.(type) {
			case []byte:
				tt.Req.Body = ioutil.NopCloser(bytes.NewReader(b))
			case func() io.ReadCloser:
				tt.Req.Body = b()
			default:
				t.Fatalf("Test %d: unsupported Body of %T", i, tt.Body)
			}
		}
		setBody()
		if tt.Req.Header == nil {
			tt.Req.Header = make(http.Header)
		}

		if tt.WantHeader != "" || tt.WantBody != "" {
			setBody()
			dumpHeader, dumpBody, err := DumpRequest(&tt.Req, !tt.NoBody)
			if err != nil {
				t.Errorf("DumpRequest #%d: %s", i, err)
				continue
			}
			if string(dumpHeader) != tt.WantHeader {
				t.Errorf("DumpRequest %d, expecting:\n%s\nGot:\n%s\n", i, tt.WantHeader, string(dumpHeader))
				continue
			}
			if string(dumpBody) != tt.WantBody {
				t.Errorf("DumpRequest %d, expecting:\n%s\nGot:\n%s\n", i, tt.WantBody, string(dumpBody))
				continue
			}

		}

	}
	if dg := runtime.NumGoroutine() - numg0; dg > 4 {
		buf := make([]byte, 4096)
		buf = buf[:runtime.Stack(buf, true)]
		t.Errorf("Unexpectedly large number of new goroutines: %d new: %s", dg, buf)
	}
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("Error parsing URL %q: %v", s, err))
	}
	return u
}

func mustReadRequest(s string) *http.Request {
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(s)))
	if err != nil {
		panic(err)
	}
	return req
}

var dumpResTests = []struct {
	res        *http.Response
	body       bool
	wantHeader string
	wantBody   string
}{
	{
		res: &http.Response{
			Status:        "200 OK",
			StatusCode:    200,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: 50,
			Header: http.Header{
				"Foo": []string{"Bar"},
			},
			Body: ioutil.NopCloser(strings.NewReader("foo")), // shouldn't be used
		},
		body: false, // to verify we see 50, not empty or 3.
		wantHeader: "HTTP/1.1 200 OK\r\n" +
			"Content-Length: 50\r\n" +
			"Foo: Bar\r\n",
	},

	{
		res: &http.Response{
			Status:        "200 OK",
			StatusCode:    200,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: 3,
			Body:          ioutil.NopCloser(strings.NewReader("foo")),
		},
		body: true,
		wantHeader: "HTTP/1.1 200 OK\r\n" +
			"Content-Length: 3\r\n",
		wantBody: `foo`,
	},

	{
		res: &http.Response{
			Status:           "200 OK",
			StatusCode:       200,
			Proto:            "HTTP/1.1",
			ProtoMajor:       1,
			ProtoMinor:       1,
			ContentLength:    -1,
			Body:             ioutil.NopCloser(strings.NewReader("foo")),
			TransferEncoding: []string{"chunked"},
		},
		body: true,
		wantHeader: "HTTP/1.1 200 OK\r\n" +
			"Transfer-Encoding: chunked\r\n",
		wantBody: `foo`,
	},
	{
		res: &http.Response{
			Status:        "200 OK",
			StatusCode:    200,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: 0,
			Header: http.Header{
				// To verify if headers are not filtered out.
				"Foo1": []string{"Bar1"},
				"Foo2": []string{"Bar2"},
			},
			Body: nil,
		},
		body: false, // to verify we see 0, not empty.
		wantHeader: "HTTP/1.1 200 OK\r\n" +
			"Foo1: Bar1\r\n" +
			"Foo2: Bar2\r\n" +
			"Content-Length: 0\r\n",
	},
}

func TestDumpResponse(t *testing.T) {
	for i, tt := range dumpResTests {
		dumpHeader, dumpBody, err := DumpResponse(tt.res, tt.body)
		if err != nil {
			t.Errorf("%d. DumpResponse = %v", i, err)
			continue
		}
		if string(dumpHeader) != tt.wantHeader {
			t.Errorf("%d.\nDumpResponse got:\n%s\n\nWant:\n%s\n", i, dumpHeader, tt.wantHeader)
		}
		if string(dumpBody) != tt.wantBody {
			t.Errorf("%d.\nDumpResponse got:\n%s\n\nWant:\n%s\n", i, dumpBody, tt.wantBody)
		}
	}
}
