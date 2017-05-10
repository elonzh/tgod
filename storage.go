package tgod

import (
	"github.com/Sirupsen/logrus"
	"github.com/go-tgod/tgod/tieba"
	"github.com/spf13/viper"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func SessionFromConfig() *mgo.Session {
	session, err := mgo.Dial(viper.GetString("database"))
	if err != nil {
		Logger.Fatalln(err)
	}
	return session
}

// 生成用于并发处理的存储任务, 在这里我们假设每次调用产生的会话都是新产生的, 并在任务结束时释放这次会话,
// 因为任务是并发的, 共享会话有可能会因为共享数据库连接而阻塞达不到并发的效果,
// 我们不用担心产生过多的数据库连接, 因为数据库连接是通过连接池共享的
func UpsertJob(collection *mgo.Collection, pairs ...interface{}) func() {
	return func() {
		defer collection.Database.Session.Close()
		entry := Logger.WithFields(logrus.Fields{"Collection": collection.Name, "NumItem": len(pairs) / 2})
		entry.Debugln("开始进行数据插入任务")
		bulk := collection.Bulk()
		bulk.Upsert(pairs...)
		result, err := bulk.Run()
		if err != nil {
			entry.Panicln(err)
		}
		entry.WithFields(logrus.Fields{"Matched": result.Matched, "Modified": result.Modified}).Debugln()
	}
}

func ForumUpsert(items ...tieba.Forum) func() {
	pairs := make([]interface{}, len(items)*2)
	for i, item := range items {
		selector := bson.M{"id": item.ID}
		i *= 2
		pairs[i] = selector
		pairs[i+1] = item
	}
	return UpsertJob(SessionFromConfig().DB("").C("Forum"), pairs...)
}

func ThreadUpsert(items ...tieba.Thread) func() {
	pairs := make([]interface{}, len(items)*2)
	for i, item := range items {
		selector := bson.M{"id": item.ID}
		i *= 2
		pairs[i] = selector
		pairs[i+1] = item
	}
	return UpsertJob(SessionFromConfig().DB("").C("Thread"), pairs...)
}
func UserUpsert(items ...tieba.User) func() {
	pairs := make([]interface{}, len(items)*2)
	for i, item := range items {
		selector := bson.M{"id": item.ID}
		i *= 2
		pairs[i] = selector
		pairs[i+1] = item
	}
	return UpsertJob(SessionFromConfig().DB("").C("User"), pairs...)
}
func PostUpsert(items ...tieba.Post) func() {
	pairs := make([]interface{}, len(items)*2)
	for i, item := range items {
		selector := bson.M{"id": item.ID}
		i *= 2
		pairs[i] = selector
		pairs[i+1] = item
	}
	return UpsertJob(SessionFromConfig().DB("").C("Post"), pairs...)
}
func SubPostUpsert(items ...tieba.SubPost) func() {
	pairs := make([]interface{}, len(items)*2)
	for i, item := range items {
		selector := bson.M{"id": item.ID}
		i *= 2
		pairs[i] = selector
		pairs[i+1] = item
	}
	return UpsertJob(SessionFromConfig().DB("").C("SubPost"), pairs...)
}

// 初始化数据库索引
func EnsureIndex() {
	session := SessionFromConfig()
	defer session.Close()
	db := session.DB("")
	idxs := map[string]mgo.Index{
		"Forum": {
			Name:       "Forum",
			Key:        []string{"id"},
			Unique:     true,
			DropDups:   true,
			Background: true,
			Sparse:     false,
		},
		"Thread": {
			Name:       "Thread",
			Key:        []string{"id"},
			Unique:     true,
			DropDups:   true,
			Background: true,
			Sparse:     false,
		},
		"Post": {
			Name:       "Post",
			Key:        []string{"id"},
			Unique:     true,
			DropDups:   true,
			Background: true,
			Sparse:     false,
		},
		"SubPost": {
			Name:       "SubPost",
			Key:        []string{"id"},
			Unique:     true,
			DropDups:   true,
			Background: true,
			Sparse:     false,
		},
		"User": {
			Name:       "User",
			Key:        []string{"id"},
			Unique:     true,
			DropDups:   true,
			Background: true,
			Sparse:     false,
		},
	}
	for c, idx := range idxs {
		if err := db.C(c).EnsureIndex(idx); err != nil {
			Logger.WithField("Collection", c).Panicln(err)
		}
	}
}
