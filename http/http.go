package http

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	genc "gopkg.in/h2non/gentleman.v2/context"
	genp "gopkg.in/h2non/gentleman.v2/plugin"
)

const DefaultDumpDir = "dump"

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

func RequestDumper(dir string, body bool) genp.Plugin {
	return genp.NewPhasePlugin("before dial", func(ctx *genc.Context, h genc.Handler) {
		fingerprint, ok := ctx.GetOk("FingerPrint")
		if !ok {
			h.Error(ctx, errors.New("RequestDumper: Can not get \"FingerPrint\" from context"))
			return
		}
		realDir := dir
		if realDir == "" {
			realDir = DefaultDumpDir
		}
		realDir = path.Join(realDir, fingerprint.(string))
		if err := os.MkdirAll(realDir, os.ModePerm); err != nil {
			h.Error(ctx, fmt.Errorf("RequestDumper: %s", err))
			return
		}
		dumpHeader, dumpBody, err := DumpRequest(ctx.Request, body)
		if err != nil {
			h.Error(ctx, fmt.Errorf("RequestDumper: %s", err))
			return
		}
		err = ioutil.WriteFile(path.Join(realDir, "request_header"), dumpHeader, os.ModePerm)
		if err != nil {
			h.Error(ctx, fmt.Errorf("RequestDumper: %s", err))
			return
		}
		if body {
			err = ioutil.WriteFile(path.Join(realDir, "request_body"), dumpBody, os.ModePerm)
			if err != nil {
				h.Error(ctx, fmt.Errorf("RequestDumper: %s", err))
				return
			}
		}
		h.Next(ctx)
	})
}

func ResponseDumper(dir string, body bool) genp.Plugin {
	return genp.NewResponsePlugin(func(ctx *genc.Context, h genc.Handler) {
		fingerprint, ok := ctx.GetOk("FingerPrint")
		if !ok {
			h.Error(ctx, errors.New("ResponseDumper: Can not get \"FingerPrint\" from context"))
			return
		}
		realDir := dir
		if realDir == "" {
			realDir = DefaultDumpDir
		}
		realDir = path.Join(realDir, fingerprint.(string))
		if err := os.MkdirAll(realDir, os.ModePerm); err != nil {
			h.Error(ctx, fmt.Errorf("ResponseDumper: %s", err))
			return
		}
		dumpHeader, dumpBody, err := DumpResponse(ctx.Response, body)
		if err != nil {
			h.Error(ctx, fmt.Errorf("ResponseDumper: %s", err))
			return
		}
		err = ioutil.WriteFile(path.Join(realDir, "response_header"), dumpHeader, os.ModePerm)
		if err != nil {
			h.Error(ctx, fmt.Errorf("ResponseDumper: %s", err))
			return
		}
		if body {
			err = ioutil.WriteFile(path.Join(realDir, "response_body"), dumpBody, os.ModePerm)
			if err != nil {
				h.Error(ctx, fmt.Errorf("ResponseDumper: %s", err))
				return
			}
		}
		h.Next(ctx)
	})
}
