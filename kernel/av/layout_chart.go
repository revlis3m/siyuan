// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package av

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/88250/lute/ast"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// ChartType 描述图表视图的图形类型。
type ChartType string

const (
	ChartTypeBar           ChartType = "bar"
	ChartTypeHorizontalBar ChartType = "horizontalBar"
	ChartTypeLine          ChartType = "line"
	ChartTypeDonut         ChartType = "donut"
	ChartTypeNumber        ChartType = "number"
)

// IsValid 判断图形类型是否合法，空值按柱状图显示。
func (t ChartType) IsValid() bool {
	switch t {
	case "", ChartTypeBar, ChartTypeHorizontalBar, ChartTypeLine, ChartTypeDonut, ChartTypeNumber:
		return true
	}
	return false
}

// ChartMeasure 描述图表对每个分组条目的汇总方式。
type ChartMeasure string

const (
	ChartMeasureCount   ChartMeasure = "count"
	ChartMeasureSum     ChartMeasure = "sum"
	ChartMeasureAverage ChartMeasure = "average"
	ChartMeasureMin     ChartMeasure = "min"
	ChartMeasureMax     ChartMeasure = "max"
)

// IsValid 判断汇总方式是否合法，空值按计数处理。
func (m ChartMeasure) IsValid() bool {
	switch m {
	case "", ChartMeasureCount, ChartMeasureSum, ChartMeasureAverage, ChartMeasureMin, ChartMeasureMax:
		return true
	}
	return false
}

// ChartSort 描述图表分组的排列顺序。
type ChartSort string

const (
	ChartSortDefault   ChartSort = "default"   // 选项顺序、日期顺序或数值顺序，其余字段按名称
	ChartSortValueDesc ChartSort = "valueDesc" // 按汇总值从大到小
	ChartSortValueAsc  ChartSort = "valueAsc"  // 按汇总值从小到大
	ChartSortLabelAsc  ChartSort = "labelAsc"  // 按分组名称
)

// IsValid 判断分组排序是否合法，空值按默认顺序处理。
func (s ChartSort) IsValid() bool {
	switch s {
	case "", ChartSortDefault, ChartSortValueDesc, ChartSortValueAsc, ChartSortLabelAsc:
		return true
	}
	return false
}

// ChartDateBucket 描述日期分组字段的时间粒度。
type ChartDateBucket string

const (
	ChartDateBucketDay     ChartDateBucket = "day"
	ChartDateBucketWeek    ChartDateBucket = "week" // ISO 周，周一开始
	ChartDateBucketMonth   ChartDateBucket = "month"
	ChartDateBucketQuarter ChartDateBucket = "quarter"
	ChartDateBucketYear    ChartDateBucket = "year"
)

// IsValid 判断时间粒度是否合法，空值按月处理。
func (b ChartDateBucket) IsValid() bool {
	switch b {
	case "", ChartDateBucketDay, ChartDateBucketWeek, ChartDateBucketMonth, ChartDateBucketQuarter, ChartDateBucketYear:
		return true
	}
	return false
}

const (
	ChartDefaultHeight = 360
	ChartMinHeight     = 160
	ChartMaxHeight     = 1600
	// ChartItemSampleLimit 是每个数据点返回的示例条目上限，供点击图形时列出条目。
	ChartItemSampleLimit = 10
	// chartItemSampleTotal 限制一次渲染返回的示例条目总数，避免日粒度长区间时响应过大。
	chartItemSampleTotal = 1000
	// chartMaxFilledBuckets 是显示空分组时补齐日期区间的上限，超过时只返回有条目的日期。
	chartMaxFilledBuckets = 1000
)

// ChartSettings 保存图表视图的字段绑定和显示设置，也是 setAttrViewChart 事务的完整数据。
// 字段绑定缺失或类型变化时保留绑定，由渲染器按未绑定处理。
type ChartSettings struct {
	ChartType       ChartType       `json:"chartType"`       // 图形类型
	XKeyID          string          `json:"xKeyID"`          // 分组字段
	DateBucket      ChartDateBucket `json:"dateBucket"`      // 分组字段为日期类字段时的时间粒度
	Measure         ChartMeasure    `json:"measure"`         // 汇总方式
	ValueKeyID      string          `json:"valueKeyID"`      // 非计数汇总使用的数字、汇总或模板字段
	StackKeyID      string          `json:"stackKeyID"`      // 可选的子分组字段：单选、状态或复选框
	Sort            ChartSort       `json:"sort"`            // 分组排序
	ShowEmptyGroups bool            `json:"showEmptyGroups"` // 显示没有条目的选项、日期区间和空值分组
	ShowLegend      bool            `json:"showLegend"`      // 显示图例
	ShowDataLabels  bool            `json:"showDataLabels"`  // 显示数据标签
	Height          int             `json:"height"`          // 图表高度（像素），0 使用默认高度
}

// DefaultChartSettings 返回未绑定字段的默认设置：按计数汇总的柱状图。
func DefaultChartSettings() ChartSettings {
	return ChartSettings{ChartType: ChartTypeBar, DateBucket: ChartDateBucketMonth, Measure: ChartMeasureCount,
		Sort: ChartSortDefault, ShowLegend: true, Height: ChartDefaultHeight}
}

// Validate 校验设置格式；字段绑定缺失或类型变化时保留绑定，由渲染器按未绑定处理。
func (s ChartSettings) Validate() error {
	if !s.ChartType.IsValid() {
		return fmt.Errorf("invalid chart type [%s]", s.ChartType)
	}
	if !s.Measure.IsValid() {
		return fmt.Errorf("invalid chart measure [%s]", s.Measure)
	}
	if !s.Sort.IsValid() {
		return fmt.Errorf("invalid chart sort [%s]", s.Sort)
	}
	if !s.DateBucket.IsValid() {
		return fmt.Errorf("invalid chart date bucket [%s]", s.DateBucket)
	}
	if 0 != s.Height && (ChartMinHeight > s.Height || ChartMaxHeight < s.Height) {
		return fmt.Errorf("invalid chart height [%d]", s.Height)
	}
	for _, id := range []string{s.XKeyID, s.ValueKeyID, s.StackKeyID} {
		if "" != id && !ast.IsNodeIDPattern(id) {
			return fmt.Errorf("invalid chart field [%s]", id)
		}
	}
	return nil
}

func (s *ChartSettings) remapKeyIDs(keyIDMap map[string]string) {
	if id := keyIDMap[s.XKeyID]; id != "" {
		s.XKeyID = id
	}
	if id := keyIDMap[s.ValueKeyID]; id != "" {
		s.ValueKeyID = id
	}
	if id := keyIDMap[s.StackKeyID]; id != "" {
		s.StackKeyID = id
	}
}

func (s ChartSettings) normalizedMeasure() ChartMeasure {
	if "" == s.Measure {
		return ChartMeasureCount
	}
	return s.Measure
}

func (s ChartSettings) normalizedBucket() ChartDateBucket {
	if "" == s.DateBucket {
		return ChartDateBucketMonth
	}
	return s.DateBucket
}

// LayoutChart 是图表布局，字段顺序和显隐复用行列布局，导出时按行列输出。
type LayoutChart struct {
	*LayoutTable
	Settings ChartSettings `json:"settings"`
}

// Chart 描述图表视图实例，复用表格的条目和字段；条目汇总后只返回统计数据。
type Chart struct {
	*Table
}

func (*Chart) GetType() LayoutType { return LayoutTypeChart }

// ChartData 是筛选后的条目在图表中的汇总结果。
type ChartData struct {
	// 实际使用的汇总方式；非计数汇总未绑定可用的数值字段时按计数汇总。
	Measure ChartMeasure `json:"measure"`
	// 分组字段和子分组字段的类型，未绑定或类型不支持时为空字符串。
	XType     KeyType `json:"xType"`
	StackType KeyType `json:"stackType"`
	// 按排序设置排列的分组，空值分组始终排在最后。
	Categories []*ChartCategory `json:"categories"`
	// 子分组的每个取值各一个系列；没有子分组时只有一个系列。
	Series []*ChartSeries `json:"series"`
	// 全部筛选条目的汇总值，每个条目只计一次；平均、最小或最大值没有数值时为 null。
	Total *float64 `json:"total"`
	// 筛选后的条目数。
	ItemCount int `json:"itemCount"`
}

// ChartCategory 是图表的一个分组。
type ChartCategory struct {
	Key           string              `json:"key"`                     // 分组标识：选项名、日期区间、关联条目 ID 或文本内容
	Label         string              `json:"label"`                   // 显示名称，复选框和空值分组为空字符串，由客户端本地化
	Color         string              `json:"color,omitempty"`         // 选项颜色
	ResolvedColor *AttributeViewColor `json:"resolvedColor,omitempty"` // 选项的自定义颜色
	Empty         bool                `json:"empty,omitempty"`         // 空值分组
	Start         *int64              `json:"start,omitempty"`         // 日期分组区间的开始时间（毫秒）
	Value         *float64            `json:"value"`                   // 分组内全部条目的汇总值
	Count         int                 `json:"count"`                   // 分组内的条目数
}

// ChartSeries 是图表的一个系列，数值与 Categories 一一对应。
type ChartSeries struct {
	Key           string              `json:"key"`
	Label         string              `json:"label"`
	Color         string              `json:"color,omitempty"`
	ResolvedColor *AttributeViewColor `json:"resolvedColor,omitempty"`
	Empty         bool                `json:"empty,omitempty"`
	// 各分组的汇总值；平均、最小或最大值没有数值时为 null。
	Values []*float64 `json:"values"`
	// 各分组中属于该系列的条目数。
	Counts []int `json:"counts"`
	// 各分组中属于该系列的示例条目，每个数据点最多 ChartItemSampleLimit 个，单次渲染总数有上限。
	Items [][]*ChartItem `json:"items"`
}

// ChartItem 是数据点中的示例条目。
type ChartItem struct {
	ID         string `json:"id"`         // 条目 ID
	Title      string `json:"title"`      // 主键内容
	BlockID    string `json:"blockID"`    // 绑定块 ID，游离条目为空字符串
	ValueID    string `json:"valueID"`    // 主键值 ID
	IsDetached bool   `json:"isDetached"` // 是否为游离条目
}

func NewChartView() *View {
	view := NewTableView()
	view.Name = GetAttributeViewI18n("chart")
	view.LayoutType = LayoutTypeChart
	view.Chart = &LayoutChart{LayoutTable: view.Table, Settings: DefaultChartSettings()}
	view.Table = nil
	return view
}

// NewChartSettings 选择分组字段：优先第一个状态字段，其次单选字段，再其次第一个日期类字段并按月分组。
func NewChartSettings(attrView *AttributeView) (ret ChartSettings) {
	ret = DefaultChartSettings()
	if nil == attrView {
		return
	}
	findKey := func(keyType KeyType) string {
		for _, keyValues := range attrView.KeyValues {
			if nil != keyValues && nil != keyValues.Key && keyValues.Key.Type == keyType {
				return keyValues.Key.ID
			}
		}
		return ""
	}
	for _, keyType := range append([]KeyType{KeyTypeStatus, KeyTypeSelect, KeyTypeDate}, DateKeyTypes()...) {
		if id := findKey(keyType); "" != id {
			ret.XKeyID = id
			break
		}
	}
	return
}

// IsChartGroupKeyType 判断字段能否作为图表的分组字段。
func IsChartGroupKeyType(keyType KeyType) bool {
	return HasKeyGroup(keyType, KeyGroupChartGroup)
}

// IsChartValueKeyType 判断字段能否作为求和、平均、最小或最大值的数值字段。
func IsChartValueKeyType(keyType KeyType) bool {
	return HasKeyGroup(keyType, KeyGroupChartValue)
}

// IsChartStackKeyType 判断字段能否作为图表的子分组字段。
func IsChartStackKeyType(keyType KeyType) bool {
	return HasKeyGroup(keyType, KeyGroupChartStack)
}

// ResolveChart 在筛选之后汇总条目：完整条目保存在 ChartRows 中供发布过滤重新汇总，Rows 不再返回条目。
func ResolveChart(chart *Chart) {
	if nil == chart || nil == chart.Table {
		return
	}
	chart.ChartRows = chart.Rows
	chart.Rows = []*TableRow{}
	RefreshChartData(chart.Table)
}

// RefreshChartData 按当前的 ChartRows 重新汇总图表数据。
func RefreshChartData(table *Table) {
	if nil == table {
		return
	}
	settings := DefaultChartSettings()
	if nil != table.Chart {
		settings = *table.Chart
	}
	table.ChartData = BuildChartData(table.Columns, table.ChartRows, settings, time.Local)
	table.RowCount = len(table.ChartRows)
}

// RetainChartRows 保留通过检查的条目并重新汇总，用于发布访问过滤。
func RetainChartRows(table *Table, keep func(*TableRow) bool) {
	if nil == table {
		return
	}
	rows := []*TableRow{}
	for _, row := range table.ChartRows {
		if nil != row && keep(row) {
			rows = append(rows, row)
		}
	}
	table.ChartRows = rows
	table.Rows = []*TableRow{}
	RefreshChartData(table)
}

// chartSum 按十进制精确累计数值：常见的短小数以 int64 定点累加，溢出或位数过多时转为 decimalSum。
type chartSum struct {
	mantissa int64
	scale    int
	fallback *decimalSum
}

var chartPow10 = [19]int64{1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000, 10000000000,
	100000000000, 1000000000000, 10000000000000, 100000000000000, 1000000000000000, 10000000000000000,
	100000000000000000, 1000000000000000000}

// chartDecimal 将数值的最短十进制表示拆成整数和小数位数，超出 int64 时返回 false。
func chartDecimal(value float64) (mantissa int64, scale int, ok bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	var buf [32]byte
	text := strconv.AppendFloat(buf[:0], value, 'f', -1, 64)
	negative := false
	digits := 0
	fraction := false
	for _, c := range text {
		switch {
		case '-' == c:
			negative = true
		case '.' == c:
			fraction = true
		default:
			if 18 <= digits && (0 != mantissa || '0' != c) {
				return 0, 0, false
			}
			mantissa = mantissa*10 + int64(c-'0')
			if 0 != mantissa {
				digits++
			}
			if fraction {
				scale++
			}
		}
	}
	if 18 < scale {
		return 0, 0, false
	}
	if negative {
		mantissa = -mantissa
	}
	return mantissa, scale, true
}

func chartMulPow10(value int64, exponent int) (int64, bool) {
	if 0 == exponent || 0 == value {
		return value, true
	}
	if 18 < exponent {
		return 0, false
	}
	factor := chartPow10[exponent]
	if value > math.MaxInt64/factor || value < math.MinInt64/factor {
		return 0, false
	}
	return value * factor, true
}

func (sum *chartSum) rat() *big.Rat {
	return new(big.Rat).SetFrac(big.NewInt(sum.mantissa), new(big.Int).SetInt64(chartPow10[sum.scale]))
}

func (sum *chartSum) add(value float64) {
	if nil == sum.fallback {
		if mantissa, scale, ok := chartDecimal(value); ok {
			current, target := sum.mantissa, sum.scale
			if scale > target {
				current, ok = chartMulPow10(current, scale-target)
				target = scale
			} else {
				mantissa, ok = chartMulPow10(mantissa, target-scale)
			}
			if ok && !(0 < mantissa && current > math.MaxInt64-mantissa) && !(0 > mantissa && current < math.MinInt64-mantissa) {
				sum.mantissa, sum.scale = current+mantissa, target
				return
			}
		}
		sum.fallback = &decimalSum{}
		sum.fallback.total.Set(sum.rat())
	}
	sum.fallback.add(value)
}

func (sum *chartSum) float64() float64 {
	if nil != sum.fallback {
		return sum.fallback.float64()
	}
	ret, _ := sum.rat().Float64()
	return ret
}

func (sum *chartSum) average(count int) float64 {
	if nil != sum.fallback {
		return sum.fallback.average(count)
	}
	ret, _ := sum.rat().Quo(sum.rat(), new(big.Rat).SetInt64(int64(count))).Float64()
	return ret
}

type chartAccumulator struct {
	count    int
	numbers  int
	sum      chartSum
	min, max float64
}

func (acc *chartAccumulator) add(number float64, hasNumber, needSum bool) {
	acc.count++
	if !hasNumber {
		return
	}
	if 0 == acc.numbers || number < acc.min {
		acc.min = number
	}
	if 0 == acc.numbers || number > acc.max {
		acc.max = number
	}
	acc.numbers++
	if needSum {
		acc.sum.add(number)
	}
}

func (acc *chartAccumulator) result(measure ChartMeasure) *float64 {
	var ret float64
	switch measure {
	case ChartMeasureSum:
		ret = acc.sum.float64()
	case ChartMeasureAverage:
		if 0 == acc.numbers {
			return nil
		}
		ret = acc.sum.average(acc.numbers)
	case ChartMeasureMin:
		if 0 == acc.numbers {
			return nil
		}
		ret = acc.min
	case ChartMeasureMax:
		if 0 == acc.numbers {
			return nil
		}
		ret = acc.max
	default:
		ret = float64(acc.count)
	}
	if math.IsNaN(ret) || math.IsInf(ret, 0) {
		return nil
	}
	return &ret
}

type chartPoint struct {
	acc   chartAccumulator
	items []*ChartItem
}

type chartPointKey struct {
	category, series int
}

// chartGroup 记录分组或系列在汇总过程中的排序信息。
type chartGroup struct {
	category *ChartCategory
	rank     float64
	byLabel  bool
	acc      chartAccumulator
}

type chartResolver struct {
	settings ChartSettings
	measure  ChartMeasure
	needSum  bool
	location *time.Location

	xColumn, valueColumn, stackColumn *TableColumn
	xIndex, valueIndex, stackIndex    int
	xKey, stackKey                    *Key

	groups       []*chartGroup
	groupIndex   map[string]int
	emptyGroup   int
	series       []*chartGroup
	seriesIndex  map[string]int
	emptySeries  int
	points       map[chartPointKey]*chartPoint
	total        chartAccumulator
	samples      int
	firstSeenSeq float64
}

func chartColumn(columns []*TableColumn, id string, accept func(KeyType) bool) (*TableColumn, int) {
	if "" == id {
		return nil, -1
	}
	for i, column := range columns {
		if nil != column && nil != column.BaseInstanceField && column.ID == id {
			if accept(column.Type) {
				return column, i
			}
			return nil, -1
		}
	}
	return nil, -1
}

// chartCellValue 读取条目的字段值，单元格与列顺序一致时直接按下标读取。
func chartCellValue(row *TableRow, column *TableColumn, index int) *Value {
	if nil == column {
		return nil
	}
	if 0 <= index && index < len(row.Cells) {
		if cell := row.Cells[index]; nil != cell && nil != cell.BaseValue && nil != cell.Value && cell.Value.KeyID == column.ID {
			return cell.Value
		}
	}
	return row.GetValue(column.ID)
}

// ChartNumber 读取数值字段的值：数字字段取数值，模板字段解析输出文本，汇总字段累加其中的数值。
func ChartNumber(value *Value) (float64, bool) {
	if nil == value {
		return 0, false
	}
	switch value.Type {
	case KeyTypeNumber:
		if nil != value.Number && value.Number.IsNotEmpty {
			return value.Number.Content, true
		}
	case KeyTypeTemplate:
		if nil != value.Template {
			content := strings.TrimSpace(value.Template.Content)
			if "" == content {
				return 0, false
			}
			if ret, err := strconv.ParseFloat(content, 64); nil == err {
				return ret, true
			}
			return util.Convert2Float(content)
		}
	case KeyTypeRollup:
		if nil == value.Rollup {
			return 0, false
		}
		var sum chartSum
		found := false
		for _, content := range value.Rollup.Contents {
			if number, ok := ChartNumber(content); ok {
				sum.add(number)
				found = true
			}
		}
		if found {
			return sum.float64(), true
		}
	}
	return 0, false
}

// BuildChartData 单次遍历条目，按分组和子分组汇总；多选和关联字段的条目计入其每个取值的分组。
func BuildChartData(columns []*TableColumn, rows []*TableRow, settings ChartSettings, location *time.Location) *ChartData {
	if nil == location {
		location = time.Local
	}
	r := &chartResolver{
		settings:    settings,
		measure:     settings.normalizedMeasure(),
		location:    location,
		groupIndex:  map[string]int{},
		emptyGroup:  -1,
		seriesIndex: map[string]int{},
		emptySeries: -1,
		points:      map[chartPointKey]*chartPoint{},
	}
	chartType := settings.ChartType
	r.xColumn, r.xIndex = chartColumn(columns, settings.XKeyID, IsChartGroupKeyType)
	if ChartMeasureCount != r.measure {
		r.valueColumn, r.valueIndex = chartColumn(columns, settings.ValueKeyID, IsChartValueKeyType)
		if nil == r.valueColumn {
			r.measure = ChartMeasureCount
		}
	}
	r.needSum = ChartMeasureSum == r.measure || ChartMeasureAverage == r.measure
	if ChartTypeDonut != chartType && ChartTypeNumber != chartType && settings.StackKeyID != settings.XKeyID {
		r.stackColumn, r.stackIndex = chartColumn(columns, settings.StackKeyID, IsChartStackKeyType)
	}
	ret := &ChartData{Measure: r.measure, Categories: []*ChartCategory{}, Series: []*ChartSeries{}}
	if nil != r.xColumn {
		ret.XType = r.xColumn.Type
		r.xKey = &Key{ID: r.xColumn.ID, Type: r.xColumn.Type, Options: r.xColumn.Options}
	}
	if nil != r.stackColumn {
		ret.StackType = r.stackColumn.Type
		r.stackKey = &Key{ID: r.stackColumn.ID, Type: r.stackColumn.Type, Options: r.stackColumn.Options}
	}
	if ChartTypeNumber != chartType && nil != r.xColumn && settings.ShowEmptyGroups {
		r.seedOptionGroups()
	}
	if nil == r.stackColumn {
		r.registerSeries("", "", 0, false, nil)
	} else if settings.ShowEmptyGroups {
		r.seedOptionSeries()
	}

	categoryIndexes := make([]int, 0, 4)
	for _, row := range rows {
		if nil == row {
			continue
		}
		ret.ItemCount++
		var number float64
		var hasNumber bool
		if nil != r.valueColumn {
			number, hasNumber = ChartNumber(chartCellValue(row, r.valueColumn, r.valueIndex))
		}
		r.total.add(number, hasNumber, r.needSum)
		if nil == r.xColumn || ChartTypeNumber == chartType {
			continue
		}
		categoryIndexes = r.rowCategories(row, categoryIndexes[:0])
		if 0 == len(categoryIndexes) {
			continue
		}
		series := 0
		if nil != r.stackColumn {
			series = r.rowSeries(row)
		}
		for _, category := range categoryIndexes {
			r.groups[category].acc.add(number, hasNumber, r.needSum)
			key := chartPointKey{category: category, series: series}
			point := r.points[key]
			if nil == point {
				point = &chartPoint{}
				r.points[key] = point
			}
			point.acc.add(number, hasNumber, r.needSum)
			if ChartItemSampleLimit > len(point.items) && chartItemSampleTotal > r.samples {
				point.items = append(point.items, newChartItem(row))
				r.samples++
			}
		}
	}
	ret.Total = r.total.result(r.measure)
	if nil == r.xColumn || ChartTypeNumber == chartType {
		return ret
	}
	if settings.ShowEmptyGroups && IsDateKeyType(r.xColumn.Type) {
		r.fillDateGaps()
	}

	order := r.sortedGroups()
	seriesOrder := r.sortedSeries()
	for _, index := range order {
		group := r.groups[index]
		group.category.Value = group.acc.result(r.measure)
		group.category.Count = group.acc.count
		ret.Categories = append(ret.Categories, group.category)
	}
	for _, seriesIndex := range seriesOrder {
		group := r.series[seriesIndex]
		if nil != r.stackColumn && !settings.ShowEmptyGroups && 0 == group.acc.count {
			continue
		}
		category := group.category
		series := &ChartSeries{Key: category.Key, Label: category.Label, Color: category.Color,
			ResolvedColor: category.ResolvedColor, Empty: category.Empty,
			Values: make([]*float64, len(order)), Counts: make([]int, len(order)), Items: make([][]*ChartItem, len(order))}
		for i, index := range order {
			point := r.points[chartPointKey{category: index, series: seriesIndex}]
			if nil == point {
				if ChartMeasureCount == r.measure || ChartMeasureSum == r.measure {
					zero := 0.0
					series.Values[i] = &zero
				}
				series.Items[i] = []*ChartItem{}
				continue
			}
			series.Values[i] = point.acc.result(r.measure)
			series.Counts[i] = point.acc.count
			series.Items[i] = point.items
		}
		ret.Series = append(ret.Series, series)
	}
	return ret
}

func newChartItem(row *TableRow) *ChartItem {
	ret := &ChartItem{ID: row.ID}
	block := row.GetBlockValue()
	if nil == block {
		return ret
	}
	ret.ValueID = block.ID
	ret.IsDetached = block.IsDetached
	if nil != block.Block {
		ret.Title = block.Block.Content
		if !block.IsDetached {
			ret.BlockID = block.Block.ID
		}
	}
	return ret
}

func (r *chartResolver) registerGroup(key, label string, rank float64, byLabel bool, apply func(*ChartCategory)) int {
	if index, ok := r.groupIndex[key]; ok {
		return index
	}
	category := &ChartCategory{Key: key, Label: label}
	if nil != apply {
		apply(category)
	}
	r.groups = append(r.groups, &chartGroup{category: category, rank: rank, byLabel: byLabel})
	r.groupIndex[key] = len(r.groups) - 1
	return len(r.groups) - 1
}

func (r *chartResolver) registerEmptyGroup() int {
	if 0 > r.emptyGroup {
		r.groups = append(r.groups, &chartGroup{category: &ChartCategory{Empty: true}})
		r.emptyGroup = len(r.groups) - 1
	}
	return r.emptyGroup
}

func (r *chartResolver) registerSeries(key, label string, rank float64, empty bool, apply func(*ChartCategory)) int {
	if empty {
		if 0 > r.emptySeries {
			r.series = append(r.series, &chartGroup{category: &ChartCategory{Empty: true}})
			r.emptySeries = len(r.series) - 1
		}
		return r.emptySeries
	}
	if index, ok := r.seriesIndex[key]; ok {
		return index
	}
	category := &ChartCategory{Key: key, Label: label}
	if nil != apply {
		apply(category)
	}
	r.series = append(r.series, &chartGroup{category: category, rank: rank})
	r.seriesIndex[key] = len(r.series) - 1
	return len(r.series) - 1
}

// optionRank 返回选项的默认排序序号：状态按分组再按选项顺序，单选和多选按选项顺序，已删除的选项排在后面。
func (r *chartResolver) optionRank(key *Key, name string) float64 {
	if KeyTypeStatus == key.Type {
		return float64(key.StatusOptionRank(name))
	}
	if index := key.GetOptionIndex(name); 0 <= index {
		return float64(index)
	}
	r.firstSeenSeq++
	return float64(1<<20) + r.firstSeenSeq
}

func chartOptionColor(key *Key, selected *ValueSelect) func(*ChartCategory) {
	return func(category *ChartCategory) {
		if option := key.GetOption(category.Key); nil != option {
			category.Color, category.ResolvedColor = option.Color, option.ResolvedColor
			return
		}
		if nil != selected {
			category.Color, category.ResolvedColor = selected.Color, selected.ResolvedColor
		}
	}
}

func chartOrderedOptions(key *Key) []*SelectOption {
	if KeyTypeStatus == key.Type {
		return key.StatusOrderedOptions()
	}
	return key.Options
}

// seedOptionGroups 显示空分组时预先登记全部选项、复选框取值和空值分组。
func (r *chartResolver) seedOptionGroups() {
	switch r.xColumn.Type {
	case KeyTypeSelect, KeyTypeMSelect, KeyTypeStatus:
		for _, option := range chartOrderedOptions(r.xKey) {
			if nil != option {
				r.registerGroup(option.Name, option.Name, r.optionRank(r.xKey, option.Name), false, chartOptionColor(r.xKey, nil))
			}
		}
	case KeyTypeCheckbox:
		r.registerGroup("checked", "", 0, false, nil)
		r.registerGroup("unchecked", "", 1, false, nil)
		return
	}
	r.registerEmptyGroup()
}

func (r *chartResolver) seedOptionSeries() {
	switch r.stackColumn.Type {
	case KeyTypeSelect, KeyTypeStatus:
		for _, option := range chartOrderedOptions(r.stackKey) {
			if nil != option {
				r.registerSeries(option.Name, option.Name, r.optionRank(r.stackKey, option.Name), false, chartOptionColor(r.stackKey, nil))
			}
		}
		r.registerSeries("", "", 0, true, nil)
	case KeyTypeCheckbox:
		r.registerSeries("checked", "", 0, false, nil)
		r.registerSeries("unchecked", "", 1, false, nil)
	}
}

func (r *chartResolver) rowSeries(row *TableRow) int {
	value := chartCellValue(row, r.stackColumn, r.stackIndex)
	if KeyTypeCheckbox == r.stackColumn.Type {
		if nil != value && nil != value.Checkbox && value.Checkbox.Checked {
			return r.registerSeries("checked", "", 0, false, nil)
		}
		return r.registerSeries("unchecked", "", 1, false, nil)
	}
	if nil == value || 1 > len(value.MSelect) || nil == value.MSelect[0] || "" == strings.TrimSpace(value.MSelect[0].Content) {
		return r.registerSeries("", "", 0, true, nil)
	}
	selected := value.MSelect[0]
	return r.registerSeries(selected.Content, selected.Content, r.optionRank(r.stackKey, selected.Content), false,
		chartOptionColor(r.stackKey, selected))
}

func (r *chartResolver) appendCategory(indexes []int, index int) []int {
	for _, existing := range indexes {
		if existing == index {
			return indexes
		}
	}
	return append(indexes, index)
}

// rowCategories 返回条目所属的分组，没有取值时归入空值分组；不显示空分组时忽略空值。
func (r *chartResolver) rowCategories(row *TableRow, indexes []int) []int {
	value := chartCellValue(row, r.xColumn, r.xIndex)
	switch r.xColumn.Type {
	case KeyTypeSelect, KeyTypeStatus, KeyTypeMSelect:
		if nil != value {
			for _, selected := range value.MSelect {
				if nil == selected || "" == strings.TrimSpace(selected.Content) {
					continue
				}
				indexes = r.appendCategory(indexes, r.registerGroup(selected.Content, selected.Content,
					r.optionRank(r.xKey, selected.Content), false, chartOptionColor(r.xKey, selected)))
				if KeyTypeMSelect != r.xColumn.Type {
					break
				}
			}
		}
	case KeyTypeCheckbox:
		if nil != value && nil != value.Checkbox && value.Checkbox.Checked {
			return append(indexes, r.registerGroup("checked", "", 0, false, nil))
		}
		return append(indexes, r.registerGroup("unchecked", "", 1, false, nil))
	case KeyTypeRelation:
		if nil != value && nil != value.Relation {
			if 0 < len(value.Relation.Contents) {
				for _, content := range value.Relation.Contents {
					if nil == content || "" == content.BlockID {
						continue
					}
					label := content.String(false)
					if nil != content.Block {
						label = content.Block.Content
					}
					indexes = r.appendCategory(indexes, r.registerGroup(content.BlockID, label, 0, true, nil))
				}
			} else {
				for _, id := range value.Relation.BlockIDs {
					if "" != id {
						indexes = r.appendCategory(indexes, r.registerGroup(id, id, 0, true, nil))
					}
				}
			}
		}
	case KeyTypeBlock:
		// 按主键分组时每个条目自成一组（同名条目不合并），标签为条目名称。
		label := ""
		if nil != value && nil != value.Block {
			label = strings.TrimSpace(value.Block.Content)
		}
		if "" == label {
			label = "—"
		}
		indexes = append(indexes, r.registerGroup(row.ID, label, 0, true, nil))
	case KeyTypeNumber:
		if nil != value && nil != value.Number && value.Number.IsNotEmpty {
			key := strconv.FormatFloat(value.Number.Content, 'f', -1, 64)
			label := value.Number.FormattedContent
			if "" == label {
				label = key
			}
			indexes = append(indexes, r.registerGroup(key, label, value.Number.Content, false, nil))
		}
	default:
		if IsDateKeyType(r.xColumn.Type) {
			if date, ok := chartDateValue(value); ok {
				indexes = append(indexes, r.registerDateGroup(chartDateBucketStart(date, r.settings.normalizedBucket(), r.location)))
			}
			break
		}
		if nil != value {
			content := strings.TrimSpace(value.String(false))
			if KeyTypeTemplate == value.Type && nil != value.Template {
				content = strings.TrimSpace(value.Template.Content)
			}
			if "" != content {
				indexes = append(indexes, r.registerGroup(content, content, 0, true, nil))
			}
		}
	}
	if 0 == len(indexes) && r.settings.ShowEmptyGroups {
		indexes = append(indexes, r.registerEmptyGroup())
	}
	return indexes
}

func chartDateValue(value *Value) (int64, bool) {
	if nil == value {
		return 0, false
	}
	switch value.Type {
	case KeyTypeDate:
		if nil != value.Date && value.Date.IsNotEmpty {
			return value.Date.Content, true
		}
	case KeyTypeCreated:
		if nil != value.Created && value.Created.IsNotEmpty {
			return value.Created.Content, true
		}
	case KeyTypeUpdated:
		if nil != value.Updated && value.Updated.IsNotEmpty {
			return value.Updated.Content, true
		}
	}
	return 0, false
}

func (r *chartResolver) registerDateGroup(start time.Time) int {
	label := ChartDateBucketLabel(start, r.settings.normalizedBucket())
	startMs := start.UnixMilli()
	return r.registerGroup(label, label, float64(startMs), false, func(category *ChartCategory) {
		category.Start = &startMs
	})
}

// ChartDateBucketStart 返回时间所在区间的开始时间；周按 ISO 8601 从周一开始，季度从一、四、七、十月开始。
func ChartDateBucketStart(ms int64, bucket ChartDateBucket, location *time.Location) time.Time {
	if nil == location {
		location = time.Local
	}
	return chartDateBucketStart(ms, bucket, location)
}

func chartDateBucketStart(ms int64, bucket ChartDateBucket, location *time.Location) time.Time {
	t := time.UnixMilli(ms).In(location)
	year, month, day := t.Date()
	switch bucket {
	case ChartDateBucketDay:
		return time.Date(year, month, day, 0, 0, 0, 0, location)
	case ChartDateBucketWeek:
		offset := (int(t.Weekday()) + 6) % 7
		return time.Date(year, month, day-offset, 0, 0, 0, 0, location)
	case ChartDateBucketQuarter:
		return time.Date(year, time.Month((int(month)-1)/3*3+1), 1, 0, 0, 0, 0, location)
	case ChartDateBucketYear:
		return time.Date(year, time.January, 1, 0, 0, 0, 0, location)
	default:
		return time.Date(year, month, 1, 0, 0, 0, 0, location)
	}
}

func chartNextDateBucket(start time.Time, bucket ChartDateBucket) time.Time {
	year, month, day := start.Date()
	switch bucket {
	case ChartDateBucketDay:
		return time.Date(year, month, day+1, 0, 0, 0, 0, start.Location())
	case ChartDateBucketWeek:
		return time.Date(year, month, day+7, 0, 0, 0, 0, start.Location())
	case ChartDateBucketQuarter:
		return time.Date(year, month+3, 1, 0, 0, 0, 0, start.Location())
	case ChartDateBucketYear:
		return time.Date(year+1, time.January, 1, 0, 0, 0, 0, start.Location())
	default:
		return time.Date(year, month+1, 1, 0, 0, 0, 0, start.Location())
	}
}

// ChartDateBucketLabel 返回日期区间的标识：2026-11-02、2026-W45、2026-11、2026-Q4 或 2026，客户端按语言格式化显示。
func ChartDateBucketLabel(start time.Time, bucket ChartDateBucket) string {
	switch bucket {
	case ChartDateBucketDay:
		return start.Format("2006-01-02")
	case ChartDateBucketWeek:
		year, week := start.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	case ChartDateBucketQuarter:
		return fmt.Sprintf("%04d-Q%d", start.Year(), (int(start.Month())-1)/3+1)
	case ChartDateBucketYear:
		return fmt.Sprintf("%04d", start.Year())
	default:
		return start.Format("2006-01")
	}
}

// fillDateGaps 显示空分组时补齐最早和最晚日期区间之间没有条目的区间。
func (r *chartResolver) fillDateGaps() {
	var first, last *ChartCategory
	for _, group := range r.groups {
		if nil == group.category.Start {
			continue
		}
		if nil == first || *group.category.Start < *first.Start {
			first = group.category
		}
		if nil == last || *group.category.Start > *last.Start {
			last = group.category
		}
	}
	if nil == first {
		return
	}
	bucket := r.settings.normalizedBucket()
	end := time.UnixMilli(*last.Start).In(r.location)
	var missing []time.Time
	count := 0
	for current := time.UnixMilli(*first.Start).In(r.location); !current.After(end); current = chartNextDateBucket(current, bucket) {
		count++
		if chartMaxFilledBuckets < count {
			return
		}
		if _, ok := r.groupIndex[ChartDateBucketLabel(current, bucket)]; !ok {
			missing = append(missing, current)
		}
	}
	for _, start := range missing {
		r.registerDateGroup(start)
	}
}

func chartGroupLabel(group *chartGroup) string {
	if "" != group.category.Label {
		return group.category.Label
	}
	return group.category.Key
}

func chartCompareDefault(a, b *chartGroup) bool {
	if a.byLabel || b.byLabel {
		return util.NaturalCompare(chartGroupLabel(a), chartGroupLabel(b))
	}
	return a.rank < b.rank
}

// sortedGroups 按排序设置返回分组下标，空值分组始终排在最后，汇总值相同或为空时保持默认顺序。
func (r *chartResolver) sortedGroups() []int {
	order := make([]int, len(r.groups))
	for i := range order {
		order[i] = i
	}
	values := make([]*float64, len(r.groups))
	for i, group := range r.groups {
		values[i] = group.acc.result(r.measure)
	}
	sortMode := r.settings.Sort
	sort.SliceStable(order, func(i, j int) bool {
		a, b := r.groups[order[i]], r.groups[order[j]]
		if a.category.Empty != b.category.Empty {
			return b.category.Empty
		}
		switch sortMode {
		case ChartSortValueDesc, ChartSortValueAsc:
			va, vb := values[order[i]], values[order[j]]
			if (nil == va) != (nil == vb) {
				return nil == vb
			}
			if nil != va && *va != *vb {
				if ChartSortValueDesc == sortMode {
					return *va > *vb
				}
				return *va < *vb
			}
		case ChartSortLabelAsc:
			la, lb := chartGroupLabel(a), chartGroupLabel(b)
			if la != lb {
				return util.NaturalCompare(la, lb)
			}
			return false
		}
		return chartCompareDefault(a, b)
	})
	return order
}

func (r *chartResolver) sortedSeries() []int {
	for key, point := range r.points {
		r.series[key.series].acc.count += point.acc.count
	}
	order := make([]int, len(r.series))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := r.series[order[i]], r.series[order[j]]
		if a.category.Empty != b.category.Empty {
			return b.category.Empty
		}
		return a.rank < b.rank
	})
	return order
}

func (attrView *AttributeView) ValidateChartLayouts() error {
	var validate func(*View) error
	validate = func(view *View) error {
		if nil == view {
			return nil
		}
		if LayoutTypeChart == view.LayoutType && nil == view.Chart {
			return fmt.Errorf("chart layout missing in view [%s]", view.ID)
		}
		if layout := view.Chart; nil != layout {
			if nil == layout.LayoutTable || nil == layout.BaseLayout || layout.Spec != 0 || nil != layout.Settings.Validate() {
				return fmt.Errorf("invalid chart layout in view [%s]", view.ID)
			}
			for _, column := range layout.Columns {
				if nil == column || nil == column.BaseField {
					return fmt.Errorf("invalid chart field in view [%s]", view.ID)
				}
			}
		}
		for _, group := range view.Groups {
			if err := validate(group); nil != err {
				return err
			}
		}
		return nil
	}
	for _, view := range attrView.Views {
		if err := validate(view); nil != err {
			return err
		}
	}
	return nil
}
