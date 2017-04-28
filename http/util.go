package http

import (
	"bytes"
	"crypto/sha1"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/PuerkitoBio/purell"
)

// drainBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
//
// It returns an error if the initial slurp of all bytes fails. It does not attempt
// to make the returned ReadClosers have identical error-matching behavior.
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

// 计算请求指纹
func RequestFingerprint(r *http.Request, withHeader bool) ([]byte, error) {
	var err error
	sha := sha1.New()
	io.WriteString(sha, r.Method)
	u := purell.NormalizeURL(r.URL, purell.FlagsUsuallySafeGreedy|purell.FlagSortQuery|purell.FlagRemoveFragment)
	io.WriteString(sha, u)
	if r.Body != nil {
		var body io.ReadCloser
		body, r.Body, err = drainBody(r.Body)
		if err != nil {
			return nil, err
		}
		defer body.Close()
		b, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, err
		}
		_, err = sha.Write(b)
		if err != nil {
			return nil, err
		}
	}
	if withHeader {
		_, err := io.WriteString(sha, EncodeHeader(r.Header))
		if err != nil {
			return nil, err
		}
	}
	return sha.Sum(nil), nil
}

// 对Header进行格式化, 可以用于输出Header和计算哈希
// https://tools.ietf.org/html/rfc2616#section-4.2
// The order in which header fields with differing field names are
// received itemScheduler not significant. However, it itemScheduler "good practice" to send
// general-header fields first, followed by request-header or response-
// header fields, and ending with the entity-header fields.
func EncodeHeader(h http.Header) string {
	if h == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	// 对Header的键进行排序
	sort.Strings(keys)
	for _, k := range keys {
		// 对值进行排序
		buf.WriteString(k + ":" + strings.Join(h[k], ";") + "\n")
	}
	return buf.String()
}
