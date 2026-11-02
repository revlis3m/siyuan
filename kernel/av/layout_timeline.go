// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package av

import (
	"fmt"
	"strings"
	"time"

	"github.com/88250/lute/ast"
)

// TimelineZoom 描述时间线的时间刻度。
type TimelineZoom string

const (
	TimelineZoomDay     TimelineZoom = "day"
	TimelineZoomWeek    TimelineZoom = "week"
	TimelineZoomMonth   TimelineZoom = "month"
	TimelineZoomQuarter TimelineZoom = "quarter"
)

// IsValid 判断时间刻度是否合法，空值按周显示。
func (zoom TimelineZoom) IsValid() bool {
	switch zoom {
	case "", TimelineZoomDay, TimelineZoomWeek, TimelineZoomMonth, TimelineZoomQuarter:
		return true
	}
	return false
}

// TimelineColorByConditional 表示时间线条目使用视图的条件颜色。
const TimelineColorByConditional = "conditional"

// TimelineSettings 保存时间线的字段绑定和显示设置；左侧显示的属性复用行列布局的字段显隐。
type TimelineSettings struct {
	StartKeyID string       `json:"startKeyID"` // 开始日期字段，空字符串表示未绑定
	EndKeyID   string       `json:"endKeyID"`   // 可选结束日期字段，空字符串时使用开始日期的结束日期
	Zoom       TimelineZoom `json:"zoom"`       // 时间刻度
	ColorBy    string       `json:"colorBy"`    // 空字符串不着色，选项类字段 ID 使用选项颜色，conditional 使用条件颜色
}

// Validate 校验设置格式；字段绑定缺失或类型变化时保留绑定，由渲染器按未绑定处理。
func (s TimelineSettings) Validate() error {
	if !s.Zoom.IsValid() {
		return fmt.Errorf("invalid timeline zoom [%s]", s.Zoom)
	}
	for _, id := range []string{s.StartKeyID, s.EndKeyID} {
		if "" != id && !ast.IsNodeIDPattern(id) {
			return fmt.Errorf("invalid timeline field [%s]", id)
		}
	}
	if "" != s.ColorBy && TimelineColorByConditional != s.ColorBy && !ast.IsNodeIDPattern(s.ColorBy) {
		return fmt.Errorf("invalid timeline color [%s]", s.ColorBy)
	}
	return nil
}

func (s *TimelineSettings) remapKeyIDs(keyIDMap map[string]string) {
	if id := keyIDMap[s.StartKeyID]; id != "" {
		s.StartKeyID = id
	}
	if id := keyIDMap[s.EndKeyID]; id != "" {
		s.EndKeyID = id
	}
	if id := keyIDMap[s.ColorBy]; id != "" {
		s.ColorBy = id
	}
}

type LayoutTimeline struct {
	*LayoutTable
	Settings TimelineSettings `json:"settings"`
}

// Timeline 描述时间线视图实例，复用表格的条目和字段操作。
type Timeline struct {
	*Table
}

func (*Timeline) GetType() LayoutType { return LayoutTypeTimeline }

// TimelineItem 是有开始日期的条目在时间线上的区间，时间为毫秒时间戳。
type TimelineItem struct {
	ID        string       `json:"id"`              // 条目 ID
	Start     int64        `json:"start"`           // 开始时间
	End       int64        `json:"end"`             // 结束时间，不早于开始时间；单日条目与开始时间相同
	IsNotTime bool         `json:"isNotTime"`       // 开始日期是否不含时间
	Invalid   bool         `json:"invalid"`         // 结束早于开始时按开始日期单日显示
	Title     string       `json:"title"`           // 主键内容
	BlockID   string       `json:"blockID"`         // 绑定块 ID，游离条目为空
	Color     *ValueSelect `json:"color,omitempty"` // 条目颜色，未着色时省略
}

func NewTimelineView() *View {
	view := NewTableView()
	view.Name = GetAttributeViewI18n("timeline")
	view.LayoutType = LayoutTypeTimeline
	view.Timeline = &LayoutTimeline{LayoutTable: view.Table, Settings: TimelineSettings{Zoom: TimelineZoomWeek}}
	view.Table = nil
	return view
}

// NewTimelineSettings 选择第一个日期字段作为开始日期，并优先使用状态字段、其次单选字段的选项颜色。
func NewTimelineSettings(attrView *AttributeView) (ret TimelineSettings) {
	ret.Zoom = TimelineZoomWeek
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
	for _, keyType := range append([]KeyType{KeyTypeDate}, DateKeyTypes()...) {
		if id := findKey(keyType); "" != id {
			ret.StartKeyID = id
			break
		}
	}
	for _, keyType := range []KeyType{KeyTypeStatus, KeyTypeSelect} {
		if id := findKey(keyType); "" != id {
			ret.ColorBy = id
			break
		}
	}
	return
}

// IsTimelineDateColumn 判断字段能否作为时间线的开始或结束日期。
func IsTimelineDateColumn(column *TableColumn) bool {
	return nil != column && IsDateKeyType(column.Type)
}

// timelineDate 读取条目日期字段的值；带显示模板的日期字段按模板输出的纯文本日期解析。
func timelineDate(value *Value, parse func(string, *time.Location) *ValueDate) *ValueDate {
	if nil == value {
		return nil
	}
	switch value.Type {
	case KeyTypeDate:
		if nil != parse {
			if value.HasRenderTemplate {
				return parse(value.RenderedContent, time.Local)
			}
			return nil
		}
		return value.Date
	case KeyTypeCreated:
		if nil != value.Created && value.Created.IsNotEmpty {
			return &ValueDate{Content: value.Created.Content, IsNotEmpty: true}
		}
	case KeyTypeUpdated:
		if nil != value.Updated && value.Updated.IsNotEmpty {
			return &ValueDate{Content: value.Updated.Content, IsNotEmpty: true}
		}
	}
	return nil
}

func timelineDateParser(column *TableColumn) func(string, *time.Location) *ValueDate {
	if nil != column && KeyTypeDate == column.Type && "" != strings.TrimSpace(column.RenderTemplate) {
		return newCalendarTemplateDateParser(column.DateFormat)
	}
	return nil
}

// ResolveTimelineInterval 返回条目在时间线上的区间。
// 开始日期为空而只有结束日期时按单日显示；结束字段为空时使用开始日期自带的结束日期，再缺失时为单日。
func ResolveTimelineInterval(startDate, endDate *ValueDate) (start, end int64, isNotTime, invalid, ok bool) {
	if nil == startDate {
		return
	}
	hasRange := startDate.HasEndDate && startDate.IsNotEmpty2
	switch {
	case startDate.IsNotEmpty:
		start = startDate.Content
	case hasRange:
		start = startDate.Content2
		hasRange = false
	default:
		return
	}
	ok = true
	isNotTime = startDate.IsNotTime
	end = start
	if nil != endDate && endDate.IsNotEmpty {
		end = endDate.Content
	} else if hasRange {
		end = startDate.Content2
	}
	if end < start {
		invalid = true
		end = start
	}
	return
}

// timelineItemColor 返回条目的颜色：选项类字段取第一个已选选项的颜色，conditional 取条件颜色的整条目背景。
func timelineItemColor(timeline *Timeline, row *TableRow, colorBy string) *ValueSelect {
	if "" == colorBy {
		return nil
	}
	if TimelineColorByConditional == colorBy {
		if nil != row.ConditionalColors && nil != row.ConditionalColors.Background {
			color := *row.ConditionalColors.Background
			return &color
		}
		return nil
	}
	column := timeline.GetColumn(colorBy)
	if nil == column || !IsSelectKeyType(column.Type) {
		return nil
	}
	value := row.GetValue(colorBy)
	if nil == value || 1 > len(value.MSelect) || nil == value.MSelect[0] {
		return nil
	}
	selected := value.MSelect[0]
	for _, option := range column.Options {
		if nil != option && option.Name == selected.Content {
			return &ValueSelect{Content: option.Name, Color: option.Color, ResolvedColor: option.ResolvedColor}
		}
	}
	color := *selected
	return &color
}

// ResolveTimelineRows 在筛选、排序和条件颜色之后解析条目区间，不应用条目分页。
// 有开始日期的条目保留在 Rows 中并按相同顺序生成 TimelineItems，其余条目进入 TimelineUnscheduled。
func ResolveTimelineRows(timeline *Timeline) {
	if nil == timeline || nil == timeline.Table {
		return
	}
	settings := TimelineSettings{Zoom: TimelineZoomWeek}
	if nil != timeline.Timeline {
		settings = *timeline.Timeline
	}
	rows := timeline.Rows
	timeline.Rows = []*TableRow{}
	timeline.TimelineItems = []*TimelineItem{}
	timeline.TimelineUnscheduled = []*TableRow{}
	startColumn := timeline.GetColumn(settings.StartKeyID)
	if !IsTimelineDateColumn(startColumn) {
		for _, row := range rows {
			if nil != row {
				timeline.TimelineUnscheduled = append(timeline.TimelineUnscheduled, row)
			}
		}
		timeline.RowCount = 0
		return
	}
	var endColumn *TableColumn
	if settings.EndKeyID != settings.StartKeyID {
		if column := timeline.GetColumn(settings.EndKeyID); IsTimelineDateColumn(column) {
			endColumn = column
		}
	}
	parseStart, parseEnd := timelineDateParser(startColumn), timelineDateParser(endColumn)
	for _, row := range rows {
		if nil == row {
			continue
		}
		startDate := timelineDate(row.GetValue(startColumn.ID), parseStart)
		var endDate *ValueDate
		if nil != endColumn {
			endDate = timelineDate(row.GetValue(endColumn.ID), parseEnd)
		}
		start, end, isNotTime, invalid, ok := ResolveTimelineInterval(startDate, endDate)
		if !ok {
			timeline.TimelineUnscheduled = append(timeline.TimelineUnscheduled, row)
			continue
		}
		item := &TimelineItem{ID: row.ID, Start: start, End: end, IsNotTime: isNotTime, Invalid: invalid,
			Color: timelineItemColor(timeline, row, settings.ColorBy)}
		item.Title, item.BlockID = timelineItemTitle(row)
		timeline.Rows = append(timeline.Rows, row)
		timeline.TimelineItems = append(timeline.TimelineItems, item)
	}
	timeline.RowCount = len(timeline.Rows)
}

// RetainTimelineRows 保留通过检查的条目及其区间，用于发布访问过滤。
func RetainTimelineRows(table *Table, keep func(*TableRow) bool) {
	if nil == table {
		return
	}
	kept := map[string]bool{}
	rows := []*TableRow{}
	for _, row := range table.Rows {
		if keep(row) {
			rows = append(rows, row)
			kept[row.ID] = true
		}
	}
	items := []*TimelineItem{}
	for _, item := range table.TimelineItems {
		if nil != item && kept[item.ID] {
			items = append(items, item)
		}
	}
	unscheduled := []*TableRow{}
	for _, row := range table.TimelineUnscheduled {
		if keep(row) {
			unscheduled = append(unscheduled, row)
		}
	}
	table.Rows, table.TimelineItems, table.TimelineUnscheduled = rows, items, unscheduled
	table.RowCount = len(rows)
}

func (attrView *AttributeView) ValidateTimelineLayouts() error {
	var validate func(*View) error
	validate = func(view *View) error {
		if nil == view {
			return nil
		}
		if LayoutTypeTimeline == view.LayoutType && nil == view.Timeline {
			return fmt.Errorf("timeline layout missing in view [%s]", view.ID)
		}
		if layout := view.Timeline; nil != layout {
			if nil == layout.LayoutTable || nil == layout.BaseLayout || layout.Spec != 0 || nil != layout.Settings.Validate() {
				return fmt.Errorf("invalid timeline layout in view [%s]", view.ID)
			}
			for _, column := range layout.Columns {
				if nil == column || nil == column.BaseField {
					return fmt.Errorf("invalid timeline field in view [%s]", view.ID)
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

// RefreshTimelineItemTitles 按条目当前的主键值更新区间标题和绑定块，用于主键值被过滤或替换之后。
func RefreshTimelineItemTitles(table *Table) {
	if nil == table {
		return
	}
	rows := map[string]*TableRow{}
	for _, row := range table.Rows {
		if nil != row {
			rows[row.ID] = row
		}
	}
	for _, item := range table.TimelineItems {
		if nil == item {
			continue
		}
		item.Title, item.BlockID = timelineItemTitle(rows[item.ID])
	}
}

func timelineItemTitle(row *TableRow) (title, blockID string) {
	if nil == row {
		return
	}
	block := row.GetBlockValue()
	if nil == block || nil == block.Block {
		return
	}
	title = block.Block.Content
	if !block.IsDetached {
		blockID = block.Block.ID
	}
	return
}
