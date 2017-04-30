package tieba

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"

	gen "gopkg.in/h2non/gentleman.v2"
)

// 客户端信息, 不同的客户端信息相同参数下返回的数据并不相同
const (
	// 客户端类型 iOS = "1", android = "2"
	ClientType = "2"
	// 客户端版本
	ClientVersion = "7.0.0"
	// 客户端子版本, 使用mini版参数能够减少一些无用信息
	SubAppType = "mini"
)

var baseValues = url.Values{
	"_client_type":    []string{ClientType},
	"_client_version": []string{ClientVersion},
	"subapp_type":     []string{SubAppType},
}

func copyValues(origin url.Values) url.Values {
	rv := make(url.Values, len(origin))
	for k, v := range origin {
		rv[k] = v
	}
	return rv
}

// 客户端POST要带数字签名，参数按字典序排列，去掉&，加上"tiebaclient!!!"，转成UTF-8，取MD5
func sign(v url.Values) (string, string) {
	var buf, sigBuf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	// 对参数进行排序
	sort.Strings(keys)
	// 拼接参数
	for _, k := range keys {
		vs := v[k]
		prefix := url.QueryEscape(k) + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(url.QueryEscape(v))
			sigBuf.WriteString(k + "=" + v)
		}
	}
	// 添加后缀
	sigBuf.WriteString("tiebaclient!!!")
	// 计算MD5
	sum := md5.Sum(sigBuf.Bytes())
	sign := hex.EncodeToString(sum[:])
	// 添加签名
	buf.WriteString("&sign=" + sign)
	// 返回生成的请求字符串和签名
	return buf.String(), sign
}

const MaxThreadNum = 100

// 获取帖子列表
// kw 贴吧关键字
// pn 第几页
// rn 每页最大条目数量, 0<=rn<=100,
// 当有直播贴时也是包含帖子列表中的, 并且不受 rn 参数的影响, 其出现的位置不一定是第一条, 可能在置顶帖后
func ThreadListRequest(kw string, pn int, rn int) *gen.Request {
	const method = "POST"
	const urlStr = "http://c.tieba.baidu.com/c/f/frs/page"
	v := copyValues(baseValues)
	v.Set("kw", kw)
	v.Set("pn", strconv.Itoa(pn))
	v.Set("rn", strconv.Itoa(rn))
	q, _ := sign(v)

	req := DefaultRequest.Clone()
	req.Method(method)
	req.URL(urlStr)
	req.BodyString(q)
	return req
}

const MaxPostNum = 30

// 获取帖子内容
// tid 帖子ID
// pn 第几页
// rn 每页最大条目数量, 2<=rn<=30
// rn=0 Error 1989002: 加载数据失败
// rn=1 Error 29: 这个楼层可能已被删除啦，去看看其他贴子吧
// withSubPost 是否带上楼中楼
func PostListRequest(tid string, pn int, rn int, withSubPost bool) *gen.Request {
	const method = "POST"
	const urlStr = "http://c.tieba.baidu.com/c/f/pb/page"
	v := copyValues(baseValues)
	v.Set("kz", tid)
	v.Set("pn", strconv.Itoa(pn))
	v.Set("rn", strconv.Itoa(rn))
	v.Set("rn", strconv.Itoa(rn))
	if withSubPost {
		v.Set("with_floor", "1")
	} else {
		v.Set("with_floor", "0")
	}
	q, _ := sign(v)

	req := DefaultRequest.Clone()
	req.Method(method)
	req.URL(urlStr)
	req.BodyString(q)
	return req
}

// todo: 楼中楼最多返回10条
// http://tieba.baidu.com/p/comment?tid=5001186228&pid=104728558494&pn=2&t=1488468355362
// rn>=10
//func (c Client) GetSubPosts(tid int, pid int, pn int, rn int) []Post {
//
//}

//func (c Client)SearchForum(kw string, pn int, rn int)
// https://tieba.baidu.com
// get /mo/q/search/forum   word=***  pn 贴吧搜索
// get /mo/q/search/thread   word=*** pn 贴子搜索
// get /mo/q/search/user   word=*** pn 用户搜索
