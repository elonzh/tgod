package tgod

import (
	"github.com/go-tgod/tgod/tieba"
	"github.com/spf13/viper"
)

// 加载配置
func loadConfig() {
	viper.SetConfigName("tgod")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			Logger.Warnln("无没找到配置文件, 使用默认配置")
		} else {
			Logger.Fatalln("载入配置文件出错: ", err)
		}
		return
	}
	Logger.Debugln("配置文件加载成功")
}

func loadDefaultSettingsFor(v *viper.Viper) {
	v.SetDefault("database", "localhost/tgod")
	v.SetDefault("maxDownloaderConcurrency", 5)
	v.SetDefault("maxScraperConcurrency", 20)
	v.SetDefault("threadPaginate", tieba.MaxThreadNum)
	v.SetDefault("postPaginate", tieba.MaxPostNum)
}

func init() {
	loadConfig()
	loadDefaultSettingsFor(viper.GetViper())
}
