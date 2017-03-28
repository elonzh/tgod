package tgod

import (
	"testing"
)

func TestCrawler(t *testing.T) {
	crawler := NewCrawler(5)
	spider := TiebaSpider{"合肥工业大学宣城校区"}
	crawler.Start(&spider)
	crawler.Wait()
}
