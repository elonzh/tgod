package tgod

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

func ThreadListRequest(kw string, pn int, rn int, withGroup bool) *gen.Request {
	const method = "POST"
	const urlStr = "http://c.tieba.baidu.com/c/f/frs/page"
	v := copyValues(baseValues)
	v.Set("kw", kw)
	v.Set("pn", strconv.Itoa(pn))
	v.Set("rn", strconv.Itoa(rn))
	// fixme: 弄清 with_group 会对结果造成什么影响并更新测试用例
	if withGroup {
		v.Set("with_group", "1")
	} else {
		v.Set("with_group", "0")
	}
	q, _ := sign(v)

	req := gen.NewRequest()
	req.Method(method)
	req.URL(urlStr)
	req.BodyString(q)
	return req
}

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

	req := gen.NewRequest()
	req.Method(method)
	req.URL(urlStr)
	req.BodyString(q)
	return req
}

const MaxThreadNum = 100

// 获取帖子列表
// kw 贴吧关键字
// pn 第几页
// rn 每页条目数量, 0<=rn<=100

// fixme: 楼中楼最多返回10条
// http://tieba.baidu.com/p/comment?tid=5001186228&pid=104728558494&pn=2&t=1488468355362
// rn>=10
//func (c Client) GetSubPosts(tid int, pid int, pn int, rn int) []Post {
//
//}

const MaxPostNum = 30

// 获取帖子内容
// tid 帖子ID
// pn 第几页
// rn 每页条目数量, 2<=rn<=30
// rn=0 Error 1989002: 加载数据失败
// rn=1 Error 29: 这个楼层可能已被删除啦，去看看其他贴子吧
// withSubPost 是否带上楼中楼

//func (c Client)SearchForum(kw string, pn int, rn int)

// https://tieba.baidu.com
// get /mo/q/search/forum   word=***  pn 贴吧搜索
// get /mo/q/search/thread   word=*** pn 贴子搜索
// get /mo/q/search/user   word=*** pn 用户搜索

// todo: 吧务操作
// https://github.com/xfgryujk/TiebaManager/blob/master/TiebaAPI/TiebaOperate.cpp

type TiebaSpider struct {
	ForumName string
}

// 初始请求, 获取置顶帖吧最新(第一页)帖子列表
func (t *TiebaSpider) StartRequests() []*gen.Request {
	req := ThreadListRequest(t.ForumName, 1, 10, false)
	req.Context.Set("CallBack", t.ParseThreadList)
	return []*gen.Request{req}
}

// 解析帖子列表, 生成每个帖子回复列表第一页请求用于得到回帖页数进行下一步请求
func (t *TiebaSpider) ParseThreadList(res *gen.Response) []*gen.Request {
	// todo: 处理帖子列表数据
	// 解析 json, 出错时会直接 panic 而不是返回 error
	tlr := new(ThreadListResponse)
	if err := res.JSON(tlr); err != nil {
		panic(err)
	}
	if err := tlr.CheckStatus(); err != nil {
		Logger.Printf("请求贴吧<%s>第%d页帖子列表出错:%s", tlr.Forum.Name, tlr.Page.CurrentPage, err)
		return nil
	}
	Logger.Printf("%s吧>共有%d页帖子, 第%d页有%d条帖子", tlr.Forum.Name, tlr.Page.TotalPage, tlr.Page.CurrentPage, len(tlr.ThreadList))
	reqNum := len(tlr.ThreadList)
	if reqNum <= 0 {
		return nil
	} else {
		reqs := make([]*gen.Request, reqNum)
		for i, thread := range tlr.ThreadList {
			//Logger.Println(thread.Title)
			req := PostListRequest(thread.ID, 1, MaxPostNum, true)
			req.Context.Set("CallBack", t.ParsePostListPage)
			reqs[i] = req
		}
		Logger.Printf("产生了%d个请求", len(reqs))
		return reqs
	}
}

// 解析第一页回帖, 生成后序的请求
func (t *TiebaSpider) ParsePostListPage(res *gen.Response) []*gen.Request {
	// todo: 处理第一页数据
	plr := new(PostListResponse)
	if err := res.JSON(plr); err != nil {
		panic(err)
	}
	if err := plr.CheckStatus(); err != nil {
		Logger.Printf("请求帖子<%s>第%d页回帖出错:%s", plr.Thread.Title, plr.Page.CurrentPage, err)
		return nil
	}
	Logger.Printf("%s吧-帖子<%s>共有%d页回帖, 第%d页有%d条回帖", plr.Forum.Name, plr.Thread.Title, plr.Page.TotalPage, plr.Page.CurrentPage, len(plr.PostList))
	// 第一页已经得到了
	reqNum := plr.Page.TotalPage - 1
	if reqNum <= 0 {
		return nil
	} else {
		reqs := make([]*gen.Request, reqNum)
		for i := 2; i <= plr.Page.TotalPage; i++ {
			req := PostListRequest(plr.Thread.ID, i, MaxPostNum, true)
			req.Context.Set("CallBack", t.ParsePostList)
			reqs[i-2] = req
		}
		Logger.Printf("产生了%d个请求", len(reqs))
		return reqs
	}
}

// 解析后续回帖
func (t *TiebaSpider) ParsePostList(res *gen.Response) []*gen.Request {
	defer Logger.Println("退出ParsePostList")
	// todo: 处理后续回帖数据
	Logger.Println("进入ParsePostList")
	plr := new(PostListResponse)
	if err := res.JSON(plr); err != nil {
		panic(err)
	}
	Logger.Printf("%s吧帖子<%s>共有%d页回帖, 第%d页有%d条回帖", plr.Forum.Name, plr.Thread.Title, plr.Page.TotalPage, plr.Page.CurrentPage, len(plr.PostList))
	return nil
}
