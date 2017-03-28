package tgod

import (
	"net/http"
	"testing"
)

func TestEncodeHeader(t *testing.T) {
	for _, test := range []struct {
		Header  http.Header
		Encoded string
	}{
		{http.Header{"User-Agent": []string{"FireFox", "Chrome"}, "Accept": []string{"Anything"}}, "Accept:Anything\nUser-Agent:FireFox;Chrome\n"},
	} {
		r := EncodeHeader(test.Header)
		if r != test.Encoded {
			t.Errorf("EncodeHeader result is %s, %s is expected", r, test.Encoded)
		}
	}

}

func TestRequestFingerprint(t *testing.T) {
	req1, _ := http.NewRequest("GET", "http://www.example.com/query?id=111&cat=222", nil)
	req2, _ := http.NewRequest("GET", "http://www.example.com/query?cat=222&id=111", nil)
	if string(RequestFingerprint(req1, false)[:]) != string(RequestFingerprint(req2, false)[:]) {
		t.Error("Unequal RequestFingerprint when querys have different order")
	}
	req1, _ = http.NewRequest("GET", "http://www.example.com/", nil)
	req2, _ = http.NewRequest("GET", "http://www.example.com/#test", nil)
	if string(RequestFingerprint(req1, false)[:]) != string(RequestFingerprint(req2, false)[:]) {
		t.Error("Unequal RequestFingerprints when url has Fragment")
	}
}
