package tieba

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/go-tgod/tgod/http"
)

var dir = path.Join(os.TempDir(), http.DefaultDumpDir)

func init() {
	err := os.RemoveAll(dir)
	if err != nil {
		Logger.Fatalln(err)
	}
	err = os.MkdirAll(dir, 0666)
	if err != nil {
		Logger.Fatalln(err)
	}
	Logger.WithField("ContentDir", dir).Infoln("Content dir was created")
}

func TestClient_GetThreadList(t *testing.T) {
	tldir := path.Join(dir, "tl")
	for i, tt := range []struct {
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
		req := ThreadListRequest(tt.kw, tt.pn, tt.rn)
		req.Use(http.Fingerprint(false))
		req.Use(http.RequestDumper(tldir, true))
		req.Use(http.ResponseDumper(tldir, true))
		res, err := req.Do()
		fg := res.Context.Get("FingerPrint").(string)
		baseMsg := fmt.Sprintf("Case=%d, FingerPrint=%s", i, fg)
		if err != nil {
			t.Error(baseMsg, err)
			continue
		}
		v := new(ThreadListResponse)
		err = res.JSON(v)
		if err != nil {
			t.Error(baseMsg, err)
			continue
		}
		err = v.CheckStatus()
		if err != nil {
			t.Error(baseMsg, err)
			continue
		}
		l := len(v.ThreadList)
		// 返回的数据长度可能会小于期望值
		if l > tt.expected {
			// 检查当与预期数据长度不匹配时是否有直播贴
			// 假设只可能只有一篇直播贴
			if l != tt.expected+1 {
				t.Errorf("%s: Length of ThreadList %d, %d expected!", baseMsg, l, tt.expected)
			}
			// 直播贴不一定是第一条数据
			for i, thread := range v.ThreadList {
				if thread.IsLivePost {
					t.Logf("%s: ThreadList has a live post with index %d", baseMsg, i)
					break
				}
				if !thread.IsTop {
					t.Errorf("%s: Length of ThreadList %d, %d expected!", baseMsg, l, tt.expected)
					break
				}
			}

		}
	}
}

func TestClient_GetPostList(t *testing.T) {
	pldir := path.Join(dir, "pl")
	for i, tt := range []struct {
		tid         string
		pn          int
		rn          int
		withSubPost bool
		expected    int
		errCode     int
	}{
		{"4003196488", 1, 0, false, 0, 1989002},
		{"4003196488", 1, 1, false, 0, 29},
		{"4003196488", 1, 2, false, 2, 0},
		{"4003196488", 1, 2, true, 2, 0},
		{"4003196488", 1, 30, false, 30, 0},
		{"4003196488", 1, 31, false, 30, 0},
	} {
		req := PostListRequest(tt.tid, tt.pn, tt.rn, false)
		req.Use(http.Fingerprint(false))
		req.Use(http.RequestDumper(pldir, true))
		req.Use(http.ResponseDumper(pldir, true))
		res, err := req.Do()
		fg := res.Context.Get("FingerPrint").(string)
		baseMsg := fmt.Sprintf("Case=%d, FingerPrint=%s", i, fg)
		if err != nil {
			t.Error(baseMsg, err)
			continue
		}
		v := new(PostListResponse)
		err = res.JSON(v)
		if err != nil {
			t.Error(baseMsg, err)
			continue
		}
		err = v.CheckStatus()
		if err != nil && v.ErrorCode != tt.errCode {
			t.Error(baseMsg, err)
			continue
		}
		l := len(v.PostList)
		if l != tt.expected {
			t.Errorf("%s: Length of PostList %d, %d expected! - args: %v", baseMsg, l, tt.expected, tt)
		}
	}
}
