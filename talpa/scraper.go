package talpa

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/jeffail/tunny"
)

type Scraper interface {
	Open()
	Close()
	Send(job func())
	NumWaitingJobs() int
	NumWorkers() int
}

type scraper struct {
	pool *tunny.WorkPool

	logger *logrus.Entry
}

var _ Scraper = (*scraper)(nil)

func (s *scraper) Open() {
	_, err := s.pool.Open()
	if err != nil {
		s.logger.Panicln(err)
	}
	s.logger.Infoln("Scraper opened")
}

func (s *scraper) Close() {
	err := s.pool.Close()
	if err != nil {
		s.logger.Panicln(err)
	}
	s.logger.Infoln("Scraper closed")
}
func (s *scraper) Send(job func()) {
	entry := s.logger.WithField("Job", fmt.Sprintf("%p", job))
	s.pool.SendWorkAsync(job, func(_ interface{}, err error) {
		if err != nil {
			s.logger.Panicln(err)
		}
		entry.Debugln("Job was finished")
	})
	entry.Debugln("Item was sent")
}
func (s *scraper) NumWaitingJobs() int {
	return int(s.pool.NumPendingAsyncJobs())
}
func (s *scraper) NumWorkers() int {
	return s.pool.NumWorkers()
}

func NewScraper(limit int) Scraper {
	if limit <= 0 {
		Logger.Fatalln("Scraper 并发必须为正整数")
	}
	scraper := new(scraper)
	scraper.pool = tunny.CreatePoolGeneric(limit)

	scraper.logger = Logger.WithField("Scraper", fmt.Sprintf("%p", scraper))
	return scraper
}
