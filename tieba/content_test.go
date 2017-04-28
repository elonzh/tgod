package tieba

import (
	"os"
	"path"
	"testing"

	"github.com/go-tgod/tgod/http"
)

func init() {
	dir := path.Join(os.TempDir(), "saver")
	err := os.RemoveAll(dir)
	if err != nil {
		Logger.Fatalln(err)
	}
	err = os.MkdirAll(dir, 0666)
	if err != nil {
		Logger.Fatalln(err)
	}
	Logger.WithField("ContentDit", dir).Infoln("Content dir was created")
	DefaultRequest.Use(http.Fingerprint(false))
	DefaultRequest.Use(http.RequestSaver(dir, true))
	DefaultRequest.Use(http.ResponseSaver(dir, true))
}

func TestClient_GetThreadList(t *testing.T) {
	for _, tt := range []struct {
		kw       string
		pn       int
		rn       int
		expected int
	}{
		{"显卡", 1, 0, 0},
		{"显卡", 1, 1, 1},
		{"显卡", 1, 100, 100},
		{"显卡", 1, 101, 100},
	} {
		req := ThreadListRequest(tt.kw, tt.pn, tt.rn, false)
		res, err := req.Do()
		if err != nil {
			t.Error(err)
			continue
		}
		v := new(ThreadListResponse)
		err = res.JSON(v)
		if err != nil {
			t.Error(err)
			continue
		}
		err = v.CheckStatus()
		if err != nil {
			t.Error(err)
			continue
		}
		l := len(v.ThreadList)
		if l != tt.expected {
			t.Errorf("%s: Length of ThreadList %d, %d expected!", res.Context.Get("FingerPrint").(string), l, tt.expected)
		}
	}
}

func TestClient_GetThread(t *testing.T) {
	for _, tt := range []struct {
		tid      string
		pn       int
		rn       int
		expected int
		errCode  string
	}{
		{"4003196488", 1, 0, 0, "1989002"},
		{"4003196488", 1, 1, 0, "29"},
		{"4003196488", 1, 2, 2, ""},
		{"4003196488", 1, 30, 30, ""},
		{"4003196488", 1, 31, 30, ""},
	} {
		req := PostListRequest(tt.tid, tt.pn, tt.rn, false)
		res, err := req.Do()
		if err != nil {
			t.Error(err)
			continue
		}
		v := new(PostListResponse)
		err = res.JSON(v)
		if err != nil {
			t.Error(err)
			continue
		}
		err = v.CheckStatus()
		if err != nil && v.ErrorCode != tt.errCode {
			t.Error(err)
			continue
		}
		l := len(v.PostList)
		if l != tt.expected {
			t.Errorf("%x: Length of PostList %d, %d expected! - args: %v", res.Context.Get("FingerPrint").(string), l, tt.expected, tt)
		}
	}
}
