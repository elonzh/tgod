package tgod

import (
	"strings"
	"testing"
)

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
		l := len(v.ThreadList)
		if l != tt.expected {
			t.Errorf("Length of ThreadList is %d, %d expected!", l, tt.expected)
		}
	}
}

func TestClient_GetThread(t *testing.T) {
	for _, tt := range []struct {
		tid      string
		pn       int
		rn       int
		expected int
		error    string
	}{
		{"5003590732", 1, 0, 0, "Error 1989002"},
		{"5003590732", 1, 1, 0, "Error 29"},
		{"5003590732", 1, 2, 2, ""},
		{"5003590732", 1, 30, 30, ""},
		{"5003590732", 1, 31, 30, ""},
	} {
		req := PostListRequest(tt.tid, tt.pn, tt.rn, false)
		res, err := req.Do()
		if err != nil {
			t.Error(err)
			continue
		}
		v := new(PostListResponse)
		err = res.JSON(v)
		if err == nil {
			l := len(v.PostList)
			if l != tt.expected {
				t.Errorf("Length of PostList is %d, %d expected! - args: %v", l, tt.expected, tt)
			}
		} else if !strings.Contains(err.Error(), tt.error) {
			t.Errorf("%s - args: %v", err, tt)
		}
	}
}
