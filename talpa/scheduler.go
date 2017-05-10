package talpa

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/Workiva/go-datastructures/queue"
	gen "gopkg.in/h2non/gentleman.v2"
)

type baseScheduler interface {
	Dispose()
	Disposed() bool
	Len() int64
	Empty() bool
}

type RequestScheduler interface {
	baseScheduler
	Put(reqs ...*gen.Request)
	Get(number int64) []*gen.Request
}

func newRequestItem(req *gen.Request) (queue.Item, error) {
	var err error
	reqItem := new(requestItem)
	reqItem.Req = req
	pri := req.Context.Get("Priority")
	if pri == nil {
		reqItem.Priority = 0
	} else {
		func() {
			// 处理Priority类型断言出错
			defer func() {
				pv := recover()
				if pv != nil {
					err = fmt.Errorf("Priority type itemScheduler %T, int itemScheduler expected.", pri)
				}
			}()
			reqItem.Priority = pri.(int)
		}()
	}
	if err != nil {
		return reqItem, err
	}
	return reqItem, nil
}

type requestItem struct {
	Req      *gen.Request
	Priority int
}

// Compare returns a bool that can be used to determine
// ordering in the priority queue.  Assuming the queue
// itemScheduler in ascending order, this should return > logic.
// Return 1 to indicate this object itemScheduler greater than the
// the other logic, 0 to indicate equality, and -1 to indicate
// less than other.
func (ri requestItem) Compare(other queue.Item) int {
	oi := other.(*requestItem)
	switch {
	case ri.Priority == oi.Priority:
		return 0 // Priority 一般不设置, 前置这个选项能减少比较次数
	case ri.Priority > oi.Priority:
		return 1
	default:
		return -1
	}
}

type requestScheduler struct {
	pq *queue.PriorityQueue

	logger *logrus.Entry
}

var _ RequestScheduler = (*requestScheduler)(nil)

func (rs *requestScheduler) Dispose() {
	rs.pq.Dispose()
	rs.logger.Infoln("RequestScheduler disposed")
}
func (rs *requestScheduler) Disposed() bool {
	return rs.pq.Disposed()
}
func (rs *requestScheduler) Len() int64 {
	return int64(rs.pq.Len())
}
func (rs *requestScheduler) Empty() bool {
	return rs.pq.Empty()
}
func (rs *requestScheduler) Put(reqs ...*gen.Request) {
	reqItems := make([]queue.Item, len(reqs))
	for i, req := range reqs {
		if req == nil {
			rs.logger.Panicln("Cann't push a nil request into queue!")
		}
		item, err := newRequestItem(req)
		if err != nil {
			rs.logger.Panicln(err)
		}
		reqItems[i] = item
	}
	// 批量入队能避免频繁地使用锁
	if err := rs.pq.Put(reqItems...); err != nil {
		rs.logger.Panicln(err)
	}
}
func (rs *requestScheduler) Get(number int64) []*gen.Request {
	items, err := rs.pq.Get(int(number))
	// err 只会是 ErrDisposed
	if err != nil {
		rs.logger.Panicln(err)
	}
	reqs := make([]*gen.Request, len(items))
	for i, item := range items {
		reqs[i] = item.(*requestItem).Req
	}
	return reqs
}

func NewRequestScheduler(hint int64) RequestScheduler {
	rs := new(requestScheduler)
	rs.pq = queue.NewPriorityQueue(int(hint), false)

	rs.logger = Logger.WithField("RequestScheduler", fmt.Sprintf("%p", rs))
	return rs
}

type JobScheduler interface {
	baseScheduler
	Put(job ...func())
	Get(number int64) []func()
}

type jobScheduler struct {
	q *queue.Queue

	logger *logrus.Entry
}

var _ JobScheduler = (*jobScheduler)(nil)

func (is *jobScheduler) Dispose() {
	is.q.Dispose()
	is.logger.Infoln("JobScheduler disposed")
}
func (is *jobScheduler) Disposed() bool {
	return is.q.Disposed()
}
func (is *jobScheduler) Len() int64 {
	return is.q.Len()
}
func (is *jobScheduler) Empty() bool {
	return is.q.Empty()
}
func (is *jobScheduler) Put(job ...func()) {
	items := make([]interface{}, len(job))
	for i, j := range job {
		items[i] = j
	}
	err := is.q.Put(items...)
	if err != nil {
		is.logger.Panicln(err)
	}
}
func (is *jobScheduler) Get(number int64) []func() {
	data, err := is.q.Get(number)
	if err != nil {
		is.logger.Panicln(err)
	}
	jobs := make([]func(), len(data))
	for i, job := range data {
		jobs[i] = job.(func())
	}
	return jobs
}

func NewJobScheduler(hint int64) JobScheduler {
	is := new(jobScheduler)
	is.q = queue.New(hint)
	is.logger = Logger.WithField("JobScheduler", fmt.Sprintf("%p", is))
	return is
}
