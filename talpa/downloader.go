package talpa

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/jeffail/tunny"
	gen "gopkg.in/h2non/gentleman.v2"
)

type Downloader interface {
	Open()
	Close()
	Fetch(req *gen.Request, h Helper)
	NumWaitingJobs() int
	NumWorkers() int
}

// 默认的请求出错处理函数
var DefaultErrBack = func(res *gen.Response) {
	Logger.Errorln(res.Error)
}

type downloadWorker struct{}

func (w downloadWorker) TunnyJob(data interface{}) interface{} {
	req := data.(*gen.Request)
	res, err := req.Do()
	if err != nil {
		errBack := DefaultErrBack
		if raw, ok := res.Context.GetOk("ErrBack"); ok {
			errBack = raw.(func(*gen.Response))
		}
		errBack(res)
		return nil
	}
	return res
}
func (w downloadWorker) TunnyReady() bool {
	return true
}

type downloader struct {
	pool *tunny.WorkPool

	logger *logrus.Entry
}

var _ Downloader = (*downloader)(nil)

func (d *downloader) Open() {
	_, err := d.pool.Open()
	if err != nil {
		d.logger.Panicln(err)
	}
	d.logger.Infoln("Downloader opened")
}
func (d *downloader) Close() {
	err := d.pool.Close()
	if err != nil {
		d.logger.Panicln(err)
	}
	d.logger.Infoln("Downloader closed")
}
func (d *downloader) Fetch(req *gen.Request, h Helper) {
	entry := d.logger.WithField("Request", fmt.Sprintf("%p", req))
	d.pool.SendWorkAsync(req, func(data interface{}, err error) {
		if err != nil {
			d.logger.Panicln(err)
		}
		res := data.(*gen.Response)
		entry.Debugln("Request was sended")
		// CallBack 不能为空
		callBack := res.Context.Get("CallBack").(func(*gen.Response, Helper))
		callBack(res, h)
		entry.Debugln("Request was processed")
	})
	entry.Debugln("Request was dispatched")
}
func (d *downloader) NumWaitingJobs() int {
	return int(d.pool.NumPendingAsyncJobs())
}
func (d *downloader) NumWorkers() int {
	return d.pool.NumWorkers()
}
func NewDownloader(limit int) Downloader {
	if limit <= 0 {
		Logger.Fatalln("Downloader 并发必须为正整数")
	}
	d := new(downloader)
	workers := make([]tunny.TunnyWorker, limit)
	for i := range workers {
		workers[i] = downloadWorker{}
	}
	d.pool = tunny.CreateCustomPool(workers)

	d.logger = Logger.WithField("Downloader", fmt.Sprintf("%p", d))
	return d
}
