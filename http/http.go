package http

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http/httputil"
	"os"
	"path"

	genc "gopkg.in/h2non/gentleman.v2/context"
	genp "gopkg.in/h2non/gentleman.v2/plugin"
)

func Fingerprint(withHeader bool) genp.Plugin {
	return genp.NewPhasePlugin("before dial", func(ctx *genc.Context, h genc.Handler) {
		fp, err := RequestFingerprint(ctx.Request, withHeader)
		if err != nil {
			h.Error(ctx, fmt.Errorf("FingerPrint: %s", err))
			return
		}
		ctx.Set("FingerPrint", fmt.Sprintf("%x", fp))
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
		realDir := dir
		if realDir == "" {
			realDir = "saver"
		}
		realDir = path.Join(realDir, fingerprint.(string))
		if err := os.MkdirAll(realDir, os.ModePerm); err != nil {
			h.Error(ctx, fmt.Errorf("RequestSaver: %s", err))
			return
		}
		dump, err := httputil.DumpRequest(ctx.Request, body)
		if err != nil {
			h.Error(ctx, fmt.Errorf("RequestSaver: %s", err))
			return
		}
		err = ioutil.WriteFile(path.Join(realDir, "request"), dump, os.ModePerm)
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
		realDir := dir
		if realDir == "" {
			realDir = "saver"
		}
		realDir = path.Join(realDir, fingerprint.(string))
		if err := os.MkdirAll(realDir, os.ModePerm); err != nil {
			h.Error(ctx, fmt.Errorf("RequestSaver: %s", err))
			return
		}
		dump, err := httputil.DumpResponse(ctx.Response, body)
		if err != nil {
			h.Error(ctx, fmt.Errorf("ResponseSaver: %s", err))
			return
		}
		err = ioutil.WriteFile(path.Join(realDir, "response"), dump, os.ModePerm)
		if err != nil {
			h.Error(ctx, fmt.Errorf("ResponseSaver: %s", err))
			return
		}
		h.Next(ctx)
	})
}
