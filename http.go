package tgod

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httputil"
	"os"
	"path"

	genc "gopkg.in/h2non/gentleman.v2/context"
	genp "gopkg.in/h2non/gentleman.v2/plugin"
)

func Fingerprint(withHeader bool) genp.Plugin {
	return genp.NewPhasePlugin("before dial", func(ctx *genc.Context, h genc.Handler) {
		r := ctx.Request
		sha := sha1.New()
		io.WriteString(sha, r.Method)
		u := CanonicalizeUrl(*r.URL, false)
		io.WriteString(sha, u.String())
		if r.Body != nil && r.ContentLength != 0 {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				h.Error(ctx, fmt.Errorf("FingerPrint: %s", err))
				return
			}
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			sha.Write(body)
		}
		if withHeader {
			io.WriteString(sha, EncodeHeader(r.Header))
		}
		ctx.Set("FingerPrint", fmt.Sprintf("%x", sha.Sum(nil)))
		h.Next(ctx)
	})
}

func RequestSaver(dir string, body bool) genp.Plugin {
	return genp.NewPhasePlugin("before dial", func(ctx *genc.Context, h genc.Handler) {
		fingerprint, ok := ctx.GetOk("FingerPrint")
		if !ok {
			h.Error(ctx, errors.New("RequestSaver: Can not get \"FingerPrint\" from context"))
			return
		}
		if dir == "" {
			dir = "saver"
		}
		dir = path.Join(dir, fingerprint.(string))
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			h.Error(ctx, fmt.Errorf("RequestSaver: %s", err))
			return
		}
		dump, err := httputil.DumpRequest(ctx.Request, body)
		if err != nil {
			h.Error(ctx, fmt.Errorf("RequestSaver: %s", err))
			return
		}
		err = ioutil.WriteFile(path.Join(dir, "request"), dump, os.ModePerm)
		if err != nil {
			h.Error(ctx, fmt.Errorf("RequestSaver: %s", err))
			return
		}
		h.Next(ctx)
	})
}

func ResponseSaver(dir string, body bool) genp.Plugin {
	return genp.NewResponsePlugin(func(ctx *genc.Context, h genc.Handler) {
		fingerprint, ok := ctx.GetOk("FingerPrint")
		if !ok {
			h.Error(ctx, errors.New("ResponseSaver: Can not get \"FingerPrint\" from context"))
			return
		}
		if dir == "" {
			dir = "saver"
		}
		dir = path.Join(dir, fingerprint.(string))
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			h.Error(ctx, fmt.Errorf("RequestSaver: %s", err))
			return
		}
		dump, err := httputil.DumpResponse(ctx.Response, body)
		if err != nil {
			h.Error(ctx, fmt.Errorf("ResponseSaver: %s", err))
			return
		}
		err = ioutil.WriteFile(path.Join(dir, "response"), dump, os.ModePerm)
		if err != nil {
			h.Error(ctx, fmt.Errorf("ResponseSaver: %s", err))
			return
		}
		h.Next(ctx)
	})
}
