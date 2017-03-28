package tgod

import (
	"fmt"

	"github.com/Workiva/go-datastructures/queue"
	gen "gopkg.in/h2non/gentleman.v2"
)

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
					err = fmt.Errorf("Priority type is %T, int is expected.", pri)
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
// is in ascending order, this should return > logic.
// Return 1 to indicate this object is greater than the
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

func NewScheduler() *Scheduler {
	sch := new(Scheduler)
	sch.pq = queue.NewPriorityQueue(4, false)
	return sch
}

// Scheduler 用于调度请求
type Scheduler struct {
	pq *queue.PriorityQueue
}

// 将请求压进队列
func (pqs *Scheduler) PushRequest(reqs ...*gen.Request) {
	reqItems := make([]queue.Item, len(reqs))
	for i, req := range reqs {
		if req == nil {
			panic("Cann't push a nil request into queue!")
		}
		item, err := newRequestItem(req)
		if err != nil {
			panic(err)
		}
		reqItems[i] = item
	}
	// 批量入队能避免频繁地使用锁
	if err := pqs.pq.Put(reqItems...); err != nil {
		panic(err)
	}
}

// 将请求从队列中取出
func (pqs *Scheduler) PopRequest(number int) []*gen.Request {
	items, err := pqs.pq.Get(number)
	// err 只会是 ErrDisposed
	if err != nil {
		panic(err)
	}
	reqs := make([]*gen.Request, len(items))
	for i, item := range items {
		reqs[i] = item.(*requestItem).Req
	}
	return reqs
}
func (pqs *Scheduler) Len() int {
	return pqs.pq.Len()
}
func (pqs *Scheduler) Empty() bool {
	return pqs.pq.Empty()
}
func (pqs *Scheduler) Close() {
	pqs.pq.Dispose()
}
func (pqs *Scheduler) IsClosed() bool {
	return pqs.pq.Disposed()
}
