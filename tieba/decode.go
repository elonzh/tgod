package tieba

import (
	"strconv"
	"strings"
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

//type TiebaUInt int
//
//// 将贴吧接口返回的无符号整数字符串转换为整数,
//// 这里的无符号整数字符串指的是数据为十进制, 实际意义非负, 可能为NAN, INF
//// 当为NAN时将值置为-1, 为INF时置为-2
//func (ti *TiebaUInt) UnmarshalJSON(data []byte) error {
//	s := strings.Trim(string(data), "\"")
//	s = strings.ToUpper(s)
//	switch s {
//	case "NAN":
//		*ti = -1
//	case "INF":
//		*ti = -2
//	default:
//		b, err := strconv.Atoi(s)
//		if err != nil {
//			return err
//		}
//		*ti = TiebaUInt(b)
//	}
//	return nil
//}
