package tieba

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func casePath(base string) ([]string, error) {
	f, err := os.Open(base)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	for i, name := range names {
		names[i] = path.Join(base, name, "response_body")
	}
	return names, nil
}

func test(t *testing.T, base string, v interface{}) {
	cases, err := casePath(base)
	if err != nil {
		t.Error(err)
	}
	for _, c := range cases {
		t.Logf("Testing case in %q", c)
		data, err := ioutil.ReadFile(c)
		if err != nil {
			t.Error(err)
		}
		err = json.Unmarshal(data, v)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestModel(t *testing.T) {
	test(t, "data_sample/pl", new(PostListResponse))
	test(t, "data_sample/tl", new(ThreadListResponse))
}
