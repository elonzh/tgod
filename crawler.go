package tgod

// todo: 支持设置停止目标, 如抓取到根据设置的最新更新日期是停止
// todo: 内容监控按规则更新, 数据抓取后台更新

import (
	"errors"
	"sync"

	"github.com/jeffail/tunny"
	gen "gopkg.in/h2non/gentleman.v2"
)

// todo: 默认的请求出错处理函数
var DefaultErrBack = func(res *gen.Response) {
	Logger.Error(res.Error)
}

type worker struct {
	Crawler *Crawler
}

func (w worker) TunnyJob(data interface{}) interface{} {
	req := data.(*gen.Request)
	//req.Use(Fingerprint(false))
	//req.Use(RequestSaver("", true))
	//req.Use(ResponseSaver("", true))
	res, err := req.Do()
	Logger.Debugln("请求已发送", req.Context.Get("FingerPrint"))
	if err != nil {
		errBack := DefaultErrBack
		if raw, ok := res.Context.GetOk("ErrBack"); ok {
			errBack = raw.(func(*gen.Response))
		}
		errBack(res)
		return nil
	}
	// CallBack 不能为空
	callBack := res.Context.Get("CallBack").(func(*gen.Response) []*gen.Request)
	reqs := callBack(res)
	if reqs == nil {
		return nil
	}
	Logger.Println("新任务组长度：", len(reqs))
	w.Crawler.sc.PushRequest(reqs...)
	return nil
}
func (w worker) TunnyReady() bool {
	return true
}

// 用于定义爬虫的接口类型
type Spider interface {
	// 生成初始的请求
	// 所有生成的请求都需要在Context设置一个"CallBack"用于对响应的解析
	// 可选的, 可以设置一个"ErrBack"用于处理发送请求时可能产生的错误
	StartRequests() []*gen.Request
}

// ErrWorkUnfinished 后台工作尚未完成时返回的错误
var ErrWorkUnfinished = errors.New("后台工作尚未完成")

// NewCrawler 初始化实例, limit 为并发限制, 为 0 表示不限制
func NewCrawler(limit int) *Crawler {
	if limit <= 0 {
		panic("Crawler 并发必须为正整数")
	}
	crawler := new(Crawler)
	workers := make([]tunny.TunnyWorker, limit)
	for i := range workers {
		workers[i] = worker{Crawler: crawler}
	}
	crawler.pool = tunny.CreateCustomPool(workers)
	crawler.sc = NewScheduler()
	crawler.done = make(chan int)
	return crawler
}

// 提供核心的爬虫工作分发机制, 只能运行一次
type Crawler struct {
	sc      *Scheduler
	pool    *tunny.WorkPool
	running bool
	done    chan int
	mutex   sync.Mutex
}

func (c *Crawler) IsRunning() bool {
	return c.running
}

func (c *Crawler) setRunning(running bool) {
	c.running = running
}

// 启动一个工作队列, 如果后台工作未完成将会启动失败并返回错误
func (c *Crawler) Start(s Spider) {
	c.mutex.Lock()
	if c.IsRunning() {
		panic(ErrWorkUnfinished)
	}
	c.setRunning(true)
	c.mutex.Unlock()
	// 添加初始请求
	c.sc.PushRequest(s.StartRequests()...)
	Logger.Debugln("初始请求已入队")
	c.pool.Open()
	Logger.Debugln("工作池已开始运行")
	go func() {
		defer c.Stop()
		// 只要请求队列不为空就继续运行
		for !c.sc.Empty() {
			// 判断是否被外部中断
			if !c.IsRunning() {
				break
			}
			Logger.Debugln("当前队列不为空：", c.sc.Len())
			// 队列不为空直接从队列中获得一个请求
			req := c.sc.PopRequest(1)
			// 同步地发送任务, 异步发送会导致请求队列一直为空, 并且不断地产生goroutineWWW
			Logger.Debugln("开始发送任务")
			if _, err := c.pool.SendWork(req[0]); err != nil {
				panic(err)
			}
			Logger.Debugln("任务处理完成")
		}
	}()
}

// 停止工作
func (c *Crawler) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	Logger.Println("开始结束工作")
	if c.IsRunning() {
		defer c.pool.Close()
		defer c.sc.Close()
		defer c.setRunning(false)
		close(c.done)
	}
	Logger.Println("爬虫工作结束")
}

// 等待工作完成
func (c *Crawler) Wait() {
	Logger.Infoln("等待爬虫工作完成...")
	// fixme: 为什么如果使用for循环将抓取工作放到后台就会卡住?
	//for c.IsRunning() {}
	<-c.done
}
