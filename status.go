package tgod

type Status struct {
	ForumID        string // 贴吧ID, 根据此ID得到贴吧的更新策略
	LatestThreadID string
	LasTime        string // 最近一篇帖子的更新时间

	Interval int
}
