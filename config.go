package tgod

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

type Config struct {
	DataDir               string // 数据存储目录
	MaxCrawlerGoroutines  int    // 爬虫请求最大并发
	MaxDatabaseGoroutines int    // 最大数据库更新最大并发

	ForumConfigs map[string]struct {
		BDUSS          string // 用于登陆的 BDUSS 值
		UpdateInterval uint   // 更新间隔, todo: 为 0 时自动调整更新频率
		Limit          uint64 // 当帖子最后更新时间戳小于这个时间戳,爬虫停止工作
	} // 每个贴吧单独的配置, 使用贴吧ID为键
}

var ExeDir string
var DefaultConfigFilePath string
var DefaultConfig Config
var GlobalConfig Config

// 加载配置
func LoadConfig(filename string) error {
	if filename == "" {
		filename = DefaultConfigFilePath
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &GlobalConfig)
	if err != nil {
		return err
	}
	return nil
}

// 保存配置
func SaveConfig(filename string) error {
	if filename == "" {
		filename = DefaultConfigFilePath
	} else if !path.IsAbs(filename) {
		filename = path.Join(ExeDir, filename)
	}
	data, err := json.MarshalIndent(GlobalConfig, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, data, 0666)
	return err
}

// 贴吧数据库以贴吧ID生成单独的数据库文件 ID.db
// 用户数据为 user.db

// todo: 存储以帖子为单位, 因为数据更新也只能也只能以帖子为单位
// 关键词数量统计
// 帖子, 关键词趋势统计
// 统计结果随更新写入数据? 还是按查询生产?

// 程序运行状态
// 存储时以贴吧ID为键
//type Status struct {
//	Latest
//	Error
//}

func init() {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	ExeDir = path.Dir(exe)
	DefaultConfigFilePath = path.Join(ExeDir, "config.json")
	DefaultConfig = Config{
		DataDir:               path.Join(ExeDir, "data"),
		MaxCrawlerGoroutines:  5,
		MaxDatabaseGoroutines: 5,
	}
	GlobalConfig = DefaultConfig
}
