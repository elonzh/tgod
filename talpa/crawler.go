package talpa

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/Sirupsen/logrus"
)

// 提供核心的爬虫工作分发机制, 只能运行一次
type Crawler struct {
	spiders          []Spider
	requestScheduler RequestScheduler
	downloader       Downloader
	jobScheduler     JobScheduler
	scraper          Scraper

	requestLoopClosed bool
	itemLoopClosed    bool
	wg                sync.WaitGroup
	stopped           chan bool

	logger *logrus.Entry
}

func (c *Crawler) Closed() bool {
	return c.requestLoopClosed && c.itemLoopClosed
}

func (c *Crawler) loopRequest() {
	c.wg.Add(1)
	c.downloader.Open()
	go func() {
		defer func() {
			c.requestScheduler.Dispose()
			c.downloader.Close()
			c.requestLoopClosed = true
			c.wg.Done()
			c.logger.Debugln("Request Loop stopped")
		}()
		// 所有请求回调共享一个helper对象, 节约内存
		h := helper{rs: c.requestScheduler, is: c.jobScheduler}
		run := true
		for run {
			select {
			case <-c.stopped:
				run = false
			default:
				if !c.requestScheduler.Empty() {
					// 异步发送会导致请求队列一直为空, 并且不断地产生等待的goroutine, 需要限制的等待任务的数量
					// 将等待任务数量与Worker数量一致确保总有任务在工作
					if c.downloader.NumWaitingJobs() < c.downloader.NumWorkers() {
						// 队列不为空直接从队列中获得一个请求
						req := c.requestScheduler.Get(1)[0]
						c.downloader.Fetch(req, &h)
					}
				} else if c.downloader.NumWaitingJobs() == 0 {
					// 调度器已为空, 也没有在等待发送的请求, 说明所有请求已处理完
					run = false
				}
			}
			c.logger.WithFields(logrus.Fields{"NumRequest": c.requestScheduler.Len(), "NumWaitingJobs": c.downloader.NumWaitingJobs()}).Debugln()
			// goroutine 是协程, 主动释放CPU避免持续占用
			runtime.Gosched()
		}
	}()
}
func (c *Crawler) loopItem() {
	c.wg.Add(1)
	// 格式化数据调度和处理
	c.scraper.Open()
	go func() {
		defer func() {
			c.jobScheduler.Dispose()
			c.scraper.Close()
			c.itemLoopClosed = true
			c.wg.Done()
			c.logger.Debugln("Item Loop stopped")
		}()
		// 即使请求任务已经结束, 只要还有未处理完的任务就继续下去
		// todo: 假设任务处理得非常快, 一瞬间就处理完了所有等待的任务但队列还没清空呢?
		run := true
		for run {
			select {
			case <-c.stopped:
				run = false
			default:
				if !c.jobScheduler.Empty() {
					if c.scraper.NumWaitingJobs() < c.scraper.NumWorkers() {
						job := c.jobScheduler.Get(1)[0]
						c.scraper.Send(job)
					}
				} else if c.requestLoopClosed && c.scraper.NumWaitingJobs() == 0 {
					// 请求处理已完成, 调度器已为空, 也没有在等待处理的任务, 说明所有任务已处理完且没有后续任务
					run = false
				}
				c.logger.WithFields(logrus.Fields{"NumItem": c.jobScheduler.Len(), "NumWaitingJobs": c.downloader.NumWaitingJobs()}).Debugln()
			}
			// goroutine 是协程, 主动释放CPU避免持续占用
			runtime.Gosched()
		}
	}()
}

// 启动一个工作队列, 如果后台工作未完成将会启动失败并返回错误
func (c *Crawler) Start() {
	// 添加初始请求
	for _, s := range c.spiders {
		c.requestScheduler.Put(s.StartRequests()...)
	}
	// 启动核心的任务调度
	c.loopRequest()
	if c.scraper != nil {
		c.loopItem()
	}
	c.logger.Infoln("Crawler started")
}

// 强制停止工作, 即使任务正在运行
func (c *Crawler) Stop() {
	close(c.stopped)
	c.Wait()
}

// 等待工作完成
func (c *Crawler) Wait() {
	c.wg.Wait()
	c.logger.Infoln("Crawler stopped")
}

// NewCrawler 初始化实例, limit 为并发限制, 为 0 表示不限制
func NewCrawler(spiders []Spider, rs RequestScheduler, d Downloader, is JobScheduler, s Scraper) *Crawler {
	crawler := new(Crawler)
	crawler.spiders = spiders
	crawler.requestScheduler = rs
	crawler.downloader = d
	if (is == nil) != (s == nil) {
		Logger.Fatalln("ItemScheduler and Scraper must be provided at the same time")
	}
	crawler.jobScheduler = is
	crawler.scraper = s

	crawler.logger = Logger.WithField("Crawler", fmt.Sprintf("%p", crawler))
	return crawler
}
