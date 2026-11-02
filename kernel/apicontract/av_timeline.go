// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package apicontract

// AVTimelineSettings 是 setAttrViewTimeline 事务的完整设置数据，也是时间线布局的持久化设置。
// 左侧列显示的属性复用视图字段的显隐设置，主键始终显示；时间线渲染不应用分组和条目分页。
type AVTimelineSettings struct {
	// 开始日期字段，可绑定 date、created 或 updated 字段；后两者只读，不支持拖动改期。
	// 带显示模板的 date 字段按模板输出的纯文本日期排布，格式与日历相同，按内核本地时区解析且不支持拖动改期。
	// 空字符串表示未绑定，此时全部条目作为待安排条目返回；字段缺失或类型变化时保留绑定并按未绑定处理。
	StartKeyID string `json:"startKeyID"`
	// 可选结束日期字段；空字符串或与开始字段相同时使用开始日期字段自带的结束日期，均缺失时按单日显示。
	EndKeyID string `json:"endKeyID"`
	// 时间刻度：day、week、month 或 quarter；空字符串按 week 显示。
	Zoom string `json:"zoom" api:"enum=|day|week|month|quarter"`
	// 条目颜色：空字符串不着色；单选、多选或状态字段 ID 使用第一个已选选项的颜色；conditional 使用视图条件颜色的整条目背景。
	ColorBy string `json:"colorBy"`
}

type AVLayoutTimeline struct {
	*AVBaseLayout
	Columns []*AVViewTableColumn `json:"columns" api:"optional,nullable"`
	// 兼容旧版行顺序；空列表与底层布局一致省略，非空列表保持原有顺序。
	RowIDs   []string           `json:"rowIds,omitempty" api:"optional,nullable"`
	Settings AVTimelineSettings `json:"settings"`
}

// AVTimelineItem 是有开始日期的条目在时间线上的区间，与 rows 顺序一致，时间均为毫秒时间戳。
// 时间按存储值返回，客户端按本地时区换算为日期；全天日期的结束日期包含当天。
type AVTimelineItem struct {
	ID    string `json:"id"`
	Start int64  `json:"start"`
	// 不早于 start；单日条目与 start 相同。
	End       int64 `json:"end"`
	IsNotTime bool  `json:"isNotTime"`
	// 结束日期早于开始日期时为 true，此时 end 与 start 相同，原值保持不变。
	Invalid bool `json:"invalid"`
	// 主键内容。
	Title string `json:"title"`
	// 绑定块 ID，游离条目为空字符串。
	BlockID string `json:"blockID"`
	// 条目颜色，未着色时省略。
	Color *AVValueSelect `json:"color,omitempty" api:"optional"`
}
