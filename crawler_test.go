package tgod

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/go-tgod/tgod/http"
	"github.com/go-tgod/tgod/talpa"
	"github.com/go-tgod/tgod/tieba"
	"github.com/spf13/viper"
)

var dir = path.Join(os.TempDir(), http.DefaultDumpDir, "crawler")

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

func TestCrawler(t *testing.T) {
	talpa.Logger.Level = logrus.InfoLevel
	Logger.Level = logrus.DebugLevel

	tieba.DefaultRequest.Use(http.Fingerprint(false))
	tieba.DefaultRequest.Use(http.RequestDumper(dir, true))
	tieba.DefaultRequest.Use(http.ResponseDumper(dir, true))

	viper.Set("database", "localhost/tgod-test")
	viper.Set("threadPaginate", 5)

	session := SessionFromConfig()
	session.DB("").DropDatabase()

	// fixme: 索引没建立完成就立即开始任务可能会导致重复键的错误
	EnsureIndex()
	time.Sleep(time.Second)

	rs := talpa.NewRequestScheduler(10)
	is := talpa.NewJobScheduler(10)
	d := talpa.NewDownloader(viper.GetInt("maxDownloaderConcurrency"))
	s := talpa.NewScraper(viper.GetInt("maxScraperConcurrency"))

	spiders := []talpa.Spider{NewTiebaSpider("程集中学")}
	crawler := talpa.NewCrawler(spiders, rs, d, is, s)
	crawler.Start()
	crawler.Wait()
}
