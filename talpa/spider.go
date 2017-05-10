package talpa

import (
	gen "gopkg.in/h2non/gentleman.v2"
)

// 用于定义爬虫的接口类型
type Spider interface {
	// 生成初始的请求
	// 所有生成的请求都需要在Context设置一个"CallBack"用于对响应的解析
	// 可选的, 可以设置一个"ErrBack"用于处理发送请求时可能产生的错误
	StartRequests() []*gen.Request
}

// 提供给响应回调的参数, 用于将新的请求或者需要处理的内容入队
type Helper interface {
	PutRequest(reqs ...*gen.Request)
	PutJob(jobs ...func())
}

var _ Helper = (*helper)(nil)

type helper struct {
	rs RequestScheduler
	is JobScheduler
}

func (h *helper) PutRequest(reqs ...*gen.Request) {
	h.rs.Put(reqs...)
}
func (h *helper) PutJob(jobs ...func()) {
	h.is.Put(jobs...)
}
