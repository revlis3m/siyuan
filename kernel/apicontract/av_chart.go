// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package apicontract

// AVChartSettings 是 setAttrViewChart 事务的完整设置数据，也是图表布局的持久化设置。
// 图表在视图筛选之后汇总条目，不应用排序、分组和条目分页；字段绑定缺失或类型变化时按未绑定处理并保留绑定。
type AVChartSettings struct {
	// 图形类型：bar、horizontalBar、line、donut 或 number；空字符串按 bar 显示。
	ChartType string `json:"chartType" api:"enum=|bar|horizontalBar|line|donut|number"`
	// 分组字段：单选、多选、状态、复选框、日期、创建时间、更新时间、关联、文本、数字、URL、邮箱、电话或模板字段。
	// 多选和关联字段的条目计入其每个取值的分组；状态选项先按状态分组再按选项顺序排列。
	XKeyID string `json:"xKeyID"`
	// 分组字段为日期类字段时的时间粒度，按内核本地时区划分；week 为 ISO 周（周一开始）。空字符串按 month 处理。
	DateBucket string `json:"dateBucket" api:"enum=|day|week|month|quarter|year"`
	// 汇总方式；空字符串按 count 处理。非计数汇总未绑定可用的数值字段时按计数汇总。
	Measure string `json:"measure" api:"enum=|count|sum|average|min|max"`
	// 求和、平均、最小或最大值使用的数字、汇总或模板字段；汇总字段累加其中的数值，模板字段解析输出文本。
	ValueKeyID string `json:"valueKeyID"`
	// 可选的子分组字段：单选、状态或复选框字段，生成堆叠柱形或多条折线；环形图和数字图忽略此设置。
	StackKeyID string `json:"stackKeyID"`
	// 分组排序：default 为选项、日期或数值顺序（其他字段按名称），valueDesc、valueAsc 按汇总值，labelAsc 按名称；空值分组始终最后。
	Sort string `json:"sort" api:"enum=|default|valueDesc|valueAsc|labelAsc"`
	// 显示没有条目的选项、日期区间之间的空区间和空值分组。
	ShowEmptyGroups bool `json:"showEmptyGroups"`
	ShowLegend      bool `json:"showLegend"`
	ShowDataLabels  bool `json:"showDataLabels"`
	// 累计值：柱形、条形和折线图的计数或求和按分组顺序显示截至该分组的合计，空值分组不计入；其他图形和汇总方式忽略此设置。
	Cumulative bool `json:"cumulative" api:"optional"`
	// 图表高度，单位为像素：0 使用默认高度 360，否则为 160 至 1600。
	Height int `json:"height"`
}

type AVLayoutChart struct {
	*AVBaseLayout
	Columns []*AVViewTableColumn `json:"columns" api:"optional,nullable"`
	// 兼容旧版行顺序；空列表与底层布局一致省略，非空列表保持原有顺序。
	RowIDs   []string        `json:"rowIds,omitempty" api:"optional,nullable"`
	Settings AVChartSettings `json:"settings"`
}

// AVChartData 是筛选后的条目在图表中的汇总结果。
type AVChartData struct {
	// 实际使用的汇总方式。
	Measure string `json:"measure" api:"enum=count|sum|average|min|max"`
	// 分组字段类型，未绑定或类型不支持时为空字符串。
	XType string `json:"xType"`
	// 子分组字段类型，未使用子分组时为空字符串。
	StackType string `json:"stackType"`
	// 按排序设置排列的分组；分组字段未绑定或图形类型为 number 时为空列表。
	Categories []*AVChartCategory `json:"categories"`
	// 子分组的每个取值各一个系列，没有子分组时只有一个 key 为空字符串的系列。
	Series []*AVChartSeries `json:"series"`
	// 全部筛选条目的汇总值，每个条目只计一次；平均、最小或最大值没有数值时为 null。
	Total *float64 `json:"total"`
	// 筛选后的条目数。
	ItemCount int `json:"itemCount"`
}

// AVChartCategory 是图表的一个分组。
type AVChartCategory struct {
	// 分组标识：选项名、日期区间标识（如 2026-11-02、2026-W45、2026-11、2026-Q4、2026）、关联条目 ID、
	// checked 或 unchecked、数值或文本内容；空值分组为空字符串。
	Key string `json:"key"`
	// 显示名称；复选框和空值分组为空字符串，由客户端本地化。
	Label         string                `json:"label"`
	Color         string                `json:"color,omitempty" api:"optional"`
	ResolvedColor *AVAttributeViewColor `json:"resolvedColor,omitempty" api:"optional"`
	Empty         bool                  `json:"empty,omitempty" api:"optional"`
	// 日期分组区间的开始时间，单位为毫秒。
	Start *int64 `json:"start,omitempty" api:"optional"`
	// 分组内全部条目的汇总值；平均、最小或最大值没有数值时为 null。
	Value *float64 `json:"value"`
	Count int      `json:"count"`
}

// AVChartSeries 是图表的一个系列，数组与 categories 一一对应。
type AVChartSeries struct {
	Key           string                `json:"key"`
	Label         string                `json:"label"`
	Color         string                `json:"color,omitempty" api:"optional"`
	ResolvedColor *AVAttributeViewColor `json:"resolvedColor,omitempty" api:"optional"`
	Empty         bool                  `json:"empty,omitempty" api:"optional"`
	// 各分组的汇总值；没有条目时计数和求和为 0，平均、最小或最大值为 null。
	Values []*float64 `json:"values"`
	Counts []int      `json:"counts"`
	// 各分组的示例条目，每个数据点最多 10 个，单次渲染最多 1000 个。
	Items [][]*AVChartItem `json:"items"`
}

type AVChartItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// 绑定块 ID，游离条目为空字符串。
	BlockID    string `json:"blockID"`
	ValueID    string `json:"valueID"`
	IsDetached bool   `json:"isDetached"`
}
