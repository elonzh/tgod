package tgod

import (
	"bytes"
	"crypto/sha1"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// 规范化Url
// 协议和域名部分不分大小写, 路径部分是否区分大小写则不一定, 要看具体网站后台是如何实现
// See Python Package: w3lib.url.canonicalize_url
func CanonicalizeUrl(u url.URL, keepFragment bool) url.URL {
	// 将query排序后重新保存
	u.RawQuery = u.Query().Encode()
	// 确保即使没有RawQuery时的一致性
	u.ForceQuery = true
	if !keepFragment {
		u.Fragment = ""
	}
	return u
}

// 计算请求指纹
func RequestFingerprint(r *http.Request, withHeader bool) []byte {
	sha := sha1.New()
	io.WriteString(sha, r.Method)
	u := CanonicalizeUrl(*r.URL, false)
	io.WriteString(sha, u.String())
	if r.Body != nil {
		body, _ := r.GetBody()
		defer body.Close()
		b, _ := ioutil.ReadAll(body)
		sha.Write(b)
	}
	if withHeader {
		io.WriteString(sha, EncodeHeader(r.Header))
	}
	return sha.Sum(nil)
}

// 对Header进行格式化, 可以用于输出Header和计算哈希
// https://tools.ietf.org/html/rfc2616#section-4.2
// The order in which header fields with differing field names are
// received is not significant. However, it is "good practice" to send
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
