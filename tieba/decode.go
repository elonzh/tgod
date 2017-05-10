package tieba

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type TiebaBool bool

// 将贴吧接口返回的布尔值字符串转换为布尔值, 空字符串转换为false
func (tb *TiebaBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" {
		*tb = false
		return nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*tb = TiebaBool(b)
	return nil
}

var _ json.Unmarshaler = (*TiebaBool)(nil)

type TiebaUInt int

// 将贴吧接口返回的无符号整数字符串转换为整数,
// 这里的无符号整数字符串指的是数据为十进制, 实际意义非负, 可能为NAN, INF
// 当为NAN时将值置为-1, 为INF时置为-2
func (ti *TiebaUInt) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	s = strings.ToUpper(s)
	switch s {
	case "NAN":
		*ti = -1
	case "INF":
		*ti = -2
	default:
		b, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*ti = TiebaUInt(b)
	}
	return nil
}

const tzOffset = 28800

// https://medium.com/coding-and-deploying-in-the-cloud/time-stamps-in-golang-abcaf581b72f
// 使用继承而不是类型别名来获得time.Time的方法, 注意我们解析得到的是UTC时间
type TiebaTime struct {
	time.Time `bson:",inline"`
}

func (tt *TiebaTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	i, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return err
	}
	tt.Time = time.Unix(i, 0).UTC()
	return nil
}

var _ json.Unmarshaler = (*TiebaTime)(nil)

func (tt TiebaTime) GetBSON() (interface{}, error) {
	if tt.IsZero() {
		return nil, nil
	}
	return tt.Time, nil
}

func (tt *TiebaTime) SetBSON(raw bson.Raw) error {
	var tm time.Time

	if err := raw.Unmarshal(&tm); err != nil {
		return err
	}

	tt.Time = tm
	return nil
}

var _ bson.Getter = (*TiebaTime)(nil)
var _ bson.Setter = (*TiebaTime)(nil)
