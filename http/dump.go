// based on "net/http/httputil/dump.go"
package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

// dumpConn is a net.Conn which writes to Writer and reads from Reader
type dumpConn struct {
	io.Writer
	io.Reader
}

func (c *dumpConn) Close() error                       { return nil }
func (c *dumpConn) LocalAddr() net.Addr                { return nil }
func (c *dumpConn) RemoteAddr() net.Addr               { return nil }
func (c *dumpConn) SetDeadline(t time.Time) error      { return nil }
func (c *dumpConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *dumpConn) SetWriteDeadline(t time.Time) error { return nil }

type neverEnding byte

func (b neverEnding) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

// delegateReader is a reader that delegates to another reader,
// once it arrives on a channel.
type delegateReader struct {
	c chan io.Reader
	r io.Reader // nil until received from c
}

func (r *delegateReader) Read(p []byte) (int, error) {
	if r.r == nil {
		r.r = <-r.c
	}
	return r.r.Read(p)
}

// Return value if nonempty, def otherwise.
func valueOrDefault(value, def string) string {
	if value != "" {
		return value
	}
	return def
}

var reqWriteExcludeHeaderDump = map[string]bool{
	"Host":              true, // not in Header map anyway
	"Transfer-Encoding": true,
	"Trailer":           true,
}

// errNoBody is a sentinel error value used by failureToReadBody so we
// can detect that the lack of body was intentional.
var errNoBody = errors.New("sentinel error value")

// failureToReadBody is a io.ReadCloser that just returns errNoBody on
// Read. It's swapped in when we don't actually wantHeader to consume
// the body, but need a non-nil one, and wantHeader to distinguish the
// error from reading the dummy body.
type failureToReadBody struct{}

func (failureToReadBody) Read([]byte) (int, error) { return 0, errNoBody }
func (failureToReadBody) Close() error             { return nil }

// based on net/http/httputil.DumpRequest but return header and body separately
// the body is original whether "Transfer-Encoding" is "chunked" or not
func DumpRequest(req *http.Request, body bool) ([]byte, []byte, error) {
	var err error
	save := req.Body
	if !body || req.Body == nil {
		req.Body = nil
	} else {
		save, req.Body, err = drainBody(req.Body)
		if err != nil {
			return nil, nil, err
		}
	}

	var headerBuf, bodyBuf bytes.Buffer

	reqURI := req.RequestURI
	if reqURI == "" {
		reqURI = req.URL.RequestURI()
	}

	fmt.Fprintf(&headerBuf, "%s %s HTTP/%d.%d\r\n", valueOrDefault(req.Method, "GET"),
		reqURI, req.ProtoMajor, req.ProtoMinor)

	absRequestURI := strings.HasPrefix(req.RequestURI, "http://") || strings.HasPrefix(req.RequestURI, "https://")
	if !absRequestURI {
		host := req.Host
		if host == "" && req.URL != nil {
			host = req.URL.Host
		}
		if host != "" {
			fmt.Fprintf(&headerBuf, "Host: %s\r\n", host)
		}
	}

	if len(req.TransferEncoding) > 0 {
		fmt.Fprintf(&headerBuf, "Transfer-Encoding: %s\r\n", strings.Join(req.TransferEncoding, ","))
	}
	if req.Close {
		fmt.Fprint(&headerBuf, "Connection: close\r\n")
	}

	err = req.Header.WriteSubset(&headerBuf, reqWriteExcludeHeaderDump)
	if err != nil {
		return nil, nil, err
	}

	if req.Body != nil {
		var dest io.Writer = &bodyBuf
		_, err = io.Copy(dest, req.Body)
	}

	req.Body = save
	if err != nil {
		return nil, nil, err
	}
	return headerBuf.Bytes(), bodyBuf.Bytes(), nil
}

// based on net/http/httputil.DumpResponse but return header and body separately
// the body is original whether "Transfer-Encoding" is "chunked" or not
func DumpResponse(resp *http.Response, body bool) ([]byte, []byte, error) {
	var err error
	headerBytes, err := httputil.DumpResponse(resp, false)
	// delete "\r\n"
	headerBytes = headerBytes[:len(headerBytes)-2]
	if !body || err != nil || resp.Body == nil {
		return headerBytes, nil, err
	}
	var bodyReader io.ReadCloser
	bodyReader, resp.Body, err = drainBody(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	defer bodyReader.Close()
	bodyBytes, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return nil, nil, err
	}
	return headerBytes, bodyBytes, nil
}
