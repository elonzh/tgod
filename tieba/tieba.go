package tieba

import (
	gen "gopkg.in/h2non/gentleman.v2"
)

// 默认的请求对象, 建立新请求对象时时调用其Clone方法, 不要直接修改此对象
var DefaultRequest *gen.Request

func init() {
	DefaultRequest = gen.NewRequest()
	DefaultRequest.SetHeader("User-Agent", "bdtb for Android "+ClientVersion)
}
