package tgod

import (
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/go-tgod/tgod/talpa"
)

func TestCrawler(t *testing.T) {
	talpa.Logger.Level = logrus.InfoLevel
	Logger.Level = logrus.DebugLevel

	GlobalConfig.Database = "localhost/tgod-test"
	session := SessionFromConfig()
	session.DB("").DropDatabase()

	// fixme: 索引没建立完成就立即开始任务可能会导致重复键的错误
	EnsureIndex()
	time.Sleep(time.Second)

	rs := talpa.NewRequestScheduler(10)
	is := talpa.NewJobScheduler(10)
	d := talpa.NewDownloader(10)
	s := talpa.NewScraper(20)

	spiders := []talpa.Spider{NewTiebaSpider("程集中学")}
	crawler := talpa.NewCrawler(spiders, rs, d, is, s)
	crawler.Start()
	crawler.Wait()
}
