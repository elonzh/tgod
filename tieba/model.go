// 包含了数据接口返回数据的结构, 忽略了一些无用数据和未知字段
package tieba

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type TextGenerator interface {
	GenerateText() string
}

type ResponseStatus struct {
	ErrorCode int    `json:"error_code,string"`
	ErrorMsg  string `json:"error_msg"`
}

func (status ResponseStatus) String() string {
	if status.ErrorCode == 0 {
		return fmt.Sprintf("Success %s: %s", status.ErrorCode, status.ErrorMsg)
	}
	return fmt.Sprintf("Error %s: %s", status.ErrorCode, status.ErrorMsg)
}

func (status ResponseStatus) CheckStatus() error {
	if status.ErrorCode == 0 {
		return nil
	}
	return fmt.Errorf("Error %s: %s", status.ErrorCode, status.ErrorMsg)
}

type Forum struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IsExists    TiebaBool `json:"is_exists" bson:"is_exists"` // 实际请求时贴吧不存在返回的是 error_code, 这个可能是贴吧被屏蔽的标志?
	Avatar      string    `json:"avatar"`
	FirstClass  string    `json:"first_class" bson:"first_class"`
	SecondClass string    `json:"second_class" bson:"second_class"`
	// 以下字段只有在请求帖子列表时得到
	Slogan    string `json:"slogan,omitempty"`
	MemberNum int    `json:"member_num,string,omitempty"  bson:"member_num"`
	ThreadNum int    `json:"thread_num,string,omitempty" bson:"thread_num"`
	PostNum   int    `json:"post_num,string,omitempty" bson:"post_num"`
}

// 帖子列表请求返回的响应有 id 和 tid 两个字段, 当帖子没有 tid 时是一个广告贴
// reply_num 包括了楼中楼数量, 为总回帖量(不包括一楼), 不能用来计算页数
// media 包含了帖子概览时的媒体文件, 这里不
type Thread struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	ReplyNum   int       `json:"reply_num,string" bson:"reply_num"`
	CreateTime TiebaTime `json:"create_time" bson:"create_time"` // 类型为时间戳, 可能不存在此字段, 比如为直播贴
	IsActivity TiebaBool `json:"is_activity" bson:"is_activity"` // 活动帖
	// 以下字段只有在请求帖子列表时得到
	AuthorID   string    `json:"author_id" bson:"author_id"`
	LastTime   TiebaTime `json:"last_time_int" bson:"last_time"` // 类型为时间戳
	ViewNum    TiebaUInt `json:"view_num" bson:"view_num"`       // 浏览量, 可能为 NAN, INF
	IsTop      TiebaBool `json:"is_top" bson:"is_top"`           // 置顶帖
	IsGood     TiebaBool `json:"is_good" bson:"is_good"`         // 精品贴
	IsNotice   TiebaBool `json:"is_notice" bson:"is_notice"`     // 通知贴
	IsBakan    TiebaBool `json:"is_bakan" bson:"is_bakan"`       // 吧刊贴
	IsVote     TiebaBool `json:"is_vote" bson:"is_vote"`         // 投票贴
	IsLivePost TiebaBool `json:"is_livepost" bson:"is_livepost"` // 直播贴, 不一定有此字段
	ForumID    string    `json:"forum_id" bson:"forum_id"`       // 所属贴吧ID, 需自行添加
}

func (t *Thread) String() string {
	return fmt.Sprintf("<Thread %s: %s>", t.ID, t.Title)
}

// 通用字段
//"page_size": "30", 请求的页大小
//"total_page": "1", 总共页数
//"current_page": "1", 当前页码
//"has_more": "0", 是否有后续页码
//"has_prev": "0", 是否有前置页码
//"offset": "0",
// 帖子列表
// cur_good_id, 意义不明
// total_count, 帖子总数
// 回帖列表
//"req_num": "30", 请求的页大小
//"total_num": "30", total_num=total_page*page_size, 没有使用价值
//"pnum": "0", 意义不明
//"tnum": "0", 意义不明
// 因此实际使用中只有 total_page, has_more, has_prev 有作用
type page struct {
	PageSize    int       `json:"page_size,string"`
	TotalPage   int       `json:"total_page,string"`
	CurrentPage int       `json:"current_page,string"`
	HasMore     TiebaBool `json:"has_more"`
	HasPrev     TiebaBool `json:"has_prev"`
}

type threadList []Thread

// 去除广告贴
func (t *threadList) UnmarshalJSON(b []byte) error {
	var oldThreadList []struct {
		Thread
		TID string // 忽略这个字段, 因为存在时与 ID 是一样的
	}
	err := json.Unmarshal(b, &oldThreadList)
	if err != nil {
		return err
	}
	for _, thread := range oldThreadList {
		if thread.TID != "" {
			*t = append(*t, thread.Thread)
		}
	}
	return nil
}

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name_show"` // 返回时有 name 字段, 但可能为空, 而且存在时应当也是与 name_show 一样的, 我们只需要一个名称标识即可
	Portrait string `json:"portrait"`  // 头像地址 http://tb.himg.baidu.com/sys/portrait/item/ + Portrait
}

// 当前请求用户, 用于检查登录状态和所在贴吧权限
type RequestUser struct {
	User
	IsLogin   TiebaBool `json:"is_login"`
	IsManager TiebaBool `json:"is_manager"`
	IsMem     TiebaBool `json:"is_mem"`
}

// 贴吧搜索接口数据结构
type ThreadListResponse struct {
	Forum       Forum       `json:"forum"`
	RequestUser RequestUser `json:"user"`
	Page        page        `json:"page"`
	ThreadList  threadList  `json:"thread_list"`
	UserList    []User      `json:"user_list"`
	ResponseStatus
}

// 楼层和楼中楼内容
// 不同的类型有不同的内容
// 帖子一楼可能会有 is_native_app: string, native_app: list 两个字段
// 0 , 文字: text
// 1 , 超链接: link
// 2 , 表情: text, c
// 3 , 图片: text, bsize, size, origin_src, cdn_src, big_cdn_src
// 4 , @: text, uid
// 5 , 视频: e_type, width, height, bsize, during_time, origin_size, text, link, src, count
// 10, 语音: during_time, voice_md5, is_sub
type Content struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty" bson:",omitempty"`
	Link       string `json:"link,omitempty" bson:",omitempty"`
	C          string `json:"c,omitempty" bson:",omitempty"`
	BSize      string `json:"bsize,omitempty" bson:",omitempty"`
	ImgSrc     string `json:"origin_src,omitempty" bson:"img_src,omitempty"`
	UID        string `json:"uid,omitempty" bson:",omitempty"`
	VoiceMD5   string `json:"voice_md5,omitempty" bson:"voice_md5,omitempty"`
	DuringTime string `json:"during_time,omitempty" bson:"during_time,omitempty"`
}

func (c Content) GenerateText() string {
	switch c.Type {
	case "0":
		return c.Text // 只需要文字回复
	case "1":
		return c.Link
	case "2":
		return ""
	case "3":
		return ""
	case "4":
		return ""
	case "5":
		return ""
	case "10":
		return ""
	default:
		Logger.Printf("Unhandled content type %2s: %v", c.Type, c)
		return ""
	}
}

type Post struct {
	ID       string    `json:"id"`
	AuthorID string    `json:"author_id" bson:"author_id"`
	Title    string    `json:"title"`
	Floor    int       `json:"floor,string"`
	Time     TiebaTime `json:"time"`
	Content  []Content `json:"content"`
	ThreadID string    `json:"thread_id" bson:"thread_id"` // 所属帖子ID, 需自行添加
}

func (p Post) GenerateText() string {
	text := ""
	// 内容数组一般很小, 没必要使用 strings.Join
	for _, c := range p.Content {
		text += c.GenerateText()
	}
	return text
}

// 将楼层和楼中楼分开存储
// 楼中楼的数据不是完整的, 后期可能会添加完整楼中楼的获取方式
// 虽然其内容类型与楼层是一样的, 但其重要性更低
type SubPost struct {
	ID       string    `json:"id"`
	AuthorID string    `json:"author_id" bson:"author_id"`
	Title    string    `json:"title"`
	Floor    int       `json:"floor,string"`
	Time     TiebaTime `json:"time"`
	Content  []Content `json:"content"`
	PostID   string    `json:"post_id" bson:"post_id"` // 楼中楼所属楼层ID, 需自行添加
}

// 没有楼回复时为 "sub_post_list: []", 否则为 "sub_post_list: {"pid": "...", sub_post_list:[...]}"
type subPostList []SubPost

func (s *subPostList) UnmarshalJSON(b []byte) error {
	type NestedSubPostList struct {
		SubPostList []SubPost `json:"sub_post_list"` // 这里使用 SubPostList 类型会造成递归解析
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t == json.Delim('[') {
		var subPostList []SubPost
		err = json.Unmarshal(b, &subPostList)
		if err != nil {
			return err
		}
		*s = subPostList
	} else {
		var n NestedSubPostList
		err = json.Unmarshal(b, &n)
		if err != nil {
			return err
		}
		*s = n.SubPostList
	}
	return nil
}

// 帖子详情接口
type PostListResponse struct {
	Forum       Forum       `json:"forum"`
	RequestUser RequestUser `json:"user"`
	Thread      Thread      `json:"thread"`
	Page        page        `json:"page"`
	PostList    []struct {
		Post
		SubPostList subPostList `json:"sub_post_list"`
	} `json:"post_list"`
	UserList []User `json:"user_list"`
	ResponseStatus
}

// 给楼层加上帖子ID, 给楼中楼加上楼层ID
func (plr *PostListResponse) UnmarshalJSON(b []byte) error {
	type aliasPostListResponse PostListResponse
	v := new(aliasPostListResponse)
	if err := json.Unmarshal(b, v); err != nil {
		return err
	}
	// 赋值是值拷贝
	for i := range v.PostList {
		v.PostList[i].ThreadID = v.Thread.ID
		for j := range v.PostList[i].SubPostList {
			v.PostList[i].SubPostList[j].PostID = v.PostList[i].ID
		}
	}
	*plr = PostListResponse(*v)
	return nil
}
