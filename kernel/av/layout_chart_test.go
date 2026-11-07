// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package av

import (
	"fmt"
	"testing"
	"time"
)

const (
	chartTestBlockKey    = "20261102000000-cblock1"
	chartTestStatusKey   = "20261102000000-cstatus"
	chartTestSelectKey   = "20261102000000-cselect"
	chartTestTagsKey     = "20261102000000-ctags01"
	chartTestNumberKey   = "20261102000000-cnumber"
	chartTestDateKey     = "20261102000000-cdate01"
	chartTestCheckKey    = "20261102000000-ccheck1"
	chartTestRelationKey = "20261102000000-crelate"
	chartTestRollupKey   = "20261102000000-crollup"
	chartTestTemplateKey = "20261102000000-ctempl1"
)

func chartTestColumns() []*TableColumn {
	column := func(id string, typ KeyType, options ...*SelectOption) *TableColumn {
		return &TableColumn{BaseInstanceField: &BaseInstanceField{ID: id, Type: typ, Options: options}}
	}
	return []*TableColumn{
		column(chartTestBlockKey, KeyTypeBlock),
		column(chartTestStatusKey, KeyTypeStatus,
			&SelectOption{Name: "Done", Color: "11", Group: StatusGroupComplete},
			&SelectOption{Name: "Not started", Color: "5", Group: StatusGroupTodo},
			&SelectOption{Name: "Doing", Color: "6", Group: StatusGroupInProgress},
			&SelectOption{Name: "Blocked", Color: "1", Group: StatusGroupInProgress}),
		column(chartTestSelectKey, KeyTypeSelect, &SelectOption{Name: "B", Color: "2"}, &SelectOption{Name: "A", Color: "3"}),
		column(chartTestTagsKey, KeyTypeMSelect, &SelectOption{Name: "red", Color: "1"}, &SelectOption{Name: "blue", Color: "7"}),
		column(chartTestNumberKey, KeyTypeNumber),
		column(chartTestDateKey, KeyTypeDate),
		column(chartTestCheckKey, KeyTypeCheckbox),
		column(chartTestRelationKey, KeyTypeRelation),
		column(chartTestRollupKey, KeyTypeRollup),
		column(chartTestTemplateKey, KeyTypeTemplate),
	}
}

var chartTestRowSeq = 0

func chartTestRow(values ...*Value) *TableRow {
	chartTestRowSeq++
	id := fmt.Sprintf("20261102%06d-row%04d", chartTestRowSeq, chartTestRowSeq)
	row := &TableRow{ID: id, Cells: []*TableCell{{BaseValue: &BaseValue{ID: id[:14] + "-v000000", ValueType: KeyTypeBlock, Value: &Value{
		ID: id[:14] + "-v000000", KeyID: chartTestBlockKey, BlockID: id, Type: KeyTypeBlock, IsDetached: true,
		Block: &ValueBlock{Content: "Item " + id}}}}}}
	for _, value := range values {
		row.Cells = append(row.Cells, &TableCell{BaseValue: &BaseValue{ID: value.KeyID + id, ValueType: value.Type, Value: value}})
	}
	return row
}

func chartStatus(name string) *Value {
	return &Value{KeyID: chartTestStatusKey, Type: KeyTypeStatus, MSelect: []*ValueSelect{{Content: name}}}
}

func chartSelect(name string) *Value {
	return &Value{KeyID: chartTestSelectKey, Type: KeyTypeSelect, MSelect: []*ValueSelect{{Content: name}}}
}

func chartTags(names ...string) *Value {
	value := &Value{KeyID: chartTestTagsKey, Type: KeyTypeMSelect}
	for _, name := range names {
		value.MSelect = append(value.MSelect, &ValueSelect{Content: name})
	}
	return value
}

func chartNumber(number float64) *Value {
	return &Value{KeyID: chartTestNumberKey, Type: KeyTypeNumber, Number: &ValueNumber{Content: number, IsNotEmpty: true}}
}

func chartDate(year int, month time.Month, day int) *Value {
	return &Value{KeyID: chartTestDateKey, Type: KeyTypeDate, Date: &ValueDate{
		Content: time.Date(year, month, day, 15, 30, 0, 0, time.UTC).UnixMilli(), IsNotEmpty: true}}
}

func chartCheck(checked bool) *Value {
	return &Value{KeyID: chartTestCheckKey, Type: KeyTypeCheckbox, Checkbox: &ValueCheckbox{Checked: checked}}
}

func chartKeys(data *ChartData) (ret []string) {
	for _, category := range data.Categories {
		if category.Empty {
			ret = append(ret, "(empty)")
		} else {
			ret = append(ret, category.Key)
		}
	}
	return
}

func chartValues(values []*float64) (ret []string) {
	for _, value := range values {
		if nil == value {
			ret = append(ret, "null")
		} else {
			ret = append(ret, fmt.Sprint(*value))
		}
	}
	return
}

func assertChartStrings(t *testing.T, name string, got, want []string) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
}

func TestChartSettingsValidate(t *testing.T) {
	valid := []ChartSettings{
		{},
		DefaultChartSettings(),
		{ChartType: ChartTypeDonut, XKeyID: chartTestStatusKey, DateBucket: ChartDateBucketQuarter, Measure: ChartMeasureMax,
			ValueKeyID: chartTestNumberKey, StackKeyID: chartTestSelectKey, Sort: ChartSortLabelAsc, Height: ChartMinHeight},
	}
	for _, settings := range valid {
		if err := settings.Validate(); nil != err {
			t.Fatalf("valid chart settings %+v rejected: %v", settings, err)
		}
	}
	invalid := []ChartSettings{
		{ChartType: "pie"},
		{Measure: "median"},
		{Sort: "random"},
		{DateBucket: "hour"},
		{Height: 100},
		{Height: ChartMaxHeight + 1},
		{XKeyID: "status"},
		{ValueKeyID: "../number"},
		{StackKeyID: "stack"},
	}
	for _, settings := range invalid {
		if err := settings.Validate(); nil == err {
			t.Fatalf("invalid chart settings %+v accepted", settings)
		}
	}
}

func TestChartDefaultSettings(t *testing.T) {
	attrView := &AttributeView{KeyValues: []*KeyValues{
		{Key: &Key{ID: chartTestBlockKey, Type: KeyTypeBlock}},
		{Key: &Key{ID: "20261102000000-ccreate", Type: KeyTypeCreated}},
		{Key: &Key{ID: chartTestDateKey, Type: KeyTypeDate}},
		{Key: &Key{ID: chartTestSelectKey, Type: KeyTypeSelect}},
		{Key: &Key{ID: chartTestStatusKey, Type: KeyTypeStatus}},
	}}
	settings := NewChartSettings(attrView)
	if settings.XKeyID != chartTestStatusKey || settings.ChartType != ChartTypeBar || settings.Measure != ChartMeasureCount ||
		settings.Sort != ChartSortDefault || settings.DateBucket != ChartDateBucketMonth || !settings.ShowLegend ||
		settings.ShowDataLabels || settings.Cumulative || settings.ShowEmptyGroups || settings.Height != ChartDefaultHeight || settings.StackKeyID != "" {
		t.Fatalf("unexpected default chart settings: %+v", settings)
	}
	attrView.KeyValues = attrView.KeyValues[:4]
	if settings = NewChartSettings(attrView); settings.XKeyID != chartTestSelectKey {
		t.Fatalf("chart must fall back to the first select field: %+v", settings)
	}
	attrView.KeyValues = attrView.KeyValues[:3]
	if settings = NewChartSettings(attrView); settings.XKeyID != chartTestDateKey || settings.DateBucket != ChartDateBucketMonth {
		t.Fatalf("chart must fall back to the first date field by month: %+v", settings)
	}
	attrView.KeyValues = attrView.KeyValues[:2]
	if settings = NewChartSettings(attrView); settings.XKeyID != "20261102000000-ccreate" {
		t.Fatalf("chart must fall back to a system date field: %+v", settings)
	}
	attrView.KeyValues = attrView.KeyValues[:1]
	if settings = NewChartSettings(attrView); settings.XKeyID != "" || nil != settings.Validate() {
		t.Fatalf("chart without groupable fields must stay unbound: %+v", settings)
	}
}

func TestChartMeasuresAndStatusOrder(t *testing.T) {
	rows := []*TableRow{
		chartTestRow(chartStatus("Done"), chartNumber(10)),
		chartTestRow(chartStatus("Done"), chartNumber(2.5)),
		chartTestRow(chartStatus("Doing"), chartNumber(-4)),
		chartTestRow(chartStatus("Not started")),
		chartTestRow(chartStatus(""), chartNumber(7)),
		chartTestRow(chartStatus("Removed"), chartNumber(0.1)),
		chartTestRow(chartStatus("Doing"), chartNumber(0.2)),
	}
	settings := DefaultChartSettings()
	settings.XKeyID = chartTestStatusKey
	data := BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	// 已删除的状态选项按待办分组排在已有待办选项之后。
	assertChartStrings(t, "status order", chartKeys(data), []string{"Not started", "Removed", "Doing", "Done"})
	if data.ItemCount != 7 || data.Measure != ChartMeasureCount || data.XType != KeyTypeStatus || 1 != len(data.Series) ||
		nil == data.Total || 7 != *data.Total {
		t.Fatalf("unexpected count summary: %+v", data)
	}
	assertChartStrings(t, "count", chartValues(data.Series[0].Values), []string{"1", "1", "2", "2"})
	if data.Categories[3].Color != "11" || data.Categories[0].Color != "5" || data.Categories[3].Label != "Done" ||
		data.Categories[3].Count != 2 || 2 != len(data.Series[0].Items[3]) || data.Series[0].Items[3][0].Title == "" ||
		!data.Series[0].Items[3][0].IsDetached || data.Series[0].Items[3][0].ValueID == "" {
		t.Fatalf("categories must carry option colors, counts and sample items: %+v", data.Categories[3])
	}

	settings.ValueKeyID = chartTestNumberKey
	for measure, want := range map[ChartMeasure][]string{
		ChartMeasureSum:     {"0", "0.1", "-3.8", "12.5"},
		ChartMeasureAverage: {"null", "0.1", "-1.9", "6.25"},
		ChartMeasureMin:     {"null", "0.1", "-4", "2.5"},
		ChartMeasureMax:     {"null", "0.1", "0.2", "10"},
	} {
		settings.Measure = measure
		data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
		assertChartStrings(t, string(measure), chartValues(data.Series[0].Values), want)
		if data.Measure != measure {
			t.Fatalf("measure %s must be kept with a number field", measure)
		}
	}
	if settings.Measure = ChartMeasureSum; nil == BuildChartData(chartTestColumns(), rows, settings, time.UTC).Total ||
		15.8 != *BuildChartData(chartTestColumns(), rows, settings, time.UTC).Total {
		t.Fatal("the total must sum every filtered item once")
	}

	settings.ShowEmptyGroups = true
	settings.Measure = ChartMeasureCount
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "empty groups", chartKeys(data), []string{"Not started", "Removed", "Doing", "Blocked", "Done", "(empty)"})
	assertChartStrings(t, "empty group values", chartValues(data.Series[0].Values), []string{"1", "1", "2", "0", "2", "1"})

	settings.ValueKeyID = chartTestSelectKey
	settings.Measure = ChartMeasureAverage
	if data = BuildChartData(chartTestColumns(), rows, settings, time.UTC); data.Measure != ChartMeasureCount {
		t.Fatal("a non-numeric value field must fall back to counting")
	}

	settings = DefaultChartSettings()
	settings.XKeyID = chartTestStatusKey
	settings.ChartType = ChartTypeNumber
	settings.Measure = ChartMeasureMax
	settings.ValueKeyID = chartTestNumberKey
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	if 0 != len(data.Categories) || 0 != len(data.Series) || nil == data.Total || 10 != *data.Total {
		t.Fatalf("number charts must only return the total: %+v", data)
	}
}

func TestChartDateBuckets(t *testing.T) {
	location := time.UTC
	cases := []struct {
		bucket ChartDateBucket
		date   time.Time
		label  string
		start  time.Time
	}{
		{ChartDateBucketDay, time.Date(2026, 3, 4, 23, 59, 0, 0, location), "2026-03-04", time.Date(2026, 3, 4, 0, 0, 0, 0, location)},
		{ChartDateBucketWeek, time.Date(2026, 12, 31, 12, 0, 0, 0, location), "2026-W53", time.Date(2026, 12, 28, 0, 0, 0, 0, location)},
		{ChartDateBucketWeek, time.Date(2027, 1, 3, 12, 0, 0, 0, location), "2026-W53", time.Date(2026, 12, 28, 0, 0, 0, 0, location)},
		{ChartDateBucketWeek, time.Date(2027, 1, 4, 0, 0, 0, 0, location), "2027-W01", time.Date(2027, 1, 4, 0, 0, 0, 0, location)},
		{ChartDateBucketWeek, time.Date(2025, 12, 29, 8, 0, 0, 0, location), "2026-W01", time.Date(2025, 12, 29, 0, 0, 0, 0, location)},
		{ChartDateBucketMonth, time.Date(2026, 2, 28, 8, 0, 0, 0, location), "2026-02", time.Date(2026, 2, 1, 0, 0, 0, 0, location)},
		{ChartDateBucketQuarter, time.Date(2026, 3, 31, 8, 0, 0, 0, location), "2026-Q1", time.Date(2026, 1, 1, 0, 0, 0, 0, location)},
		{ChartDateBucketQuarter, time.Date(2026, 5, 1, 8, 0, 0, 0, location), "2026-Q2", time.Date(2026, 4, 1, 0, 0, 0, 0, location)},
		{ChartDateBucketQuarter, time.Date(2026, 12, 31, 8, 0, 0, 0, location), "2026-Q4", time.Date(2026, 10, 1, 0, 0, 0, 0, location)},
		{ChartDateBucketYear, time.Date(2026, 7, 9, 8, 0, 0, 0, location), "2026", time.Date(2026, 1, 1, 0, 0, 0, 0, location)},
	}
	for _, test := range cases {
		start := ChartDateBucketStart(test.date.UnixMilli(), test.bucket, location)
		if !start.Equal(test.start) || ChartDateBucketLabel(start, test.bucket) != test.label {
			t.Fatalf("%s bucket of %s: got %s %s", test.bucket, test.date, start, ChartDateBucketLabel(start, test.bucket))
		}
	}

	rows := []*TableRow{
		chartTestRow(chartDate(2026, 4, 2)),
		chartTestRow(chartDate(2026, 1, 15)),
		chartTestRow(chartDate(2026, 1, 1)),
		chartTestRow(),
		chartTestRow(&Value{KeyID: chartTestDateKey, Type: KeyTypeDate, Date: &ValueDate{}}),
	}
	settings := DefaultChartSettings()
	settings.XKeyID = chartTestDateKey
	data := BuildChartData(chartTestColumns(), rows, settings, location)
	assertChartStrings(t, "months", chartKeys(data), []string{"2026-01", "2026-04"})
	assertChartStrings(t, "month counts", chartValues(data.Series[0].Values), []string{"2", "1"})
	if nil == data.Categories[1].Start || *data.Categories[1].Start != time.Date(2026, 4, 1, 0, 0, 0, 0, location).UnixMilli() {
		t.Fatalf("date categories must carry the bucket start: %+v", data.Categories[1])
	}
	settings.ShowEmptyGroups = true
	data = BuildChartData(chartTestColumns(), rows, settings, location)
	assertChartStrings(t, "filled months", chartKeys(data), []string{"2026-01", "2026-02", "2026-03", "2026-04", "(empty)"})
	assertChartStrings(t, "filled month counts", chartValues(data.Series[0].Values), []string{"2", "0", "0", "1", "2"})
	settings.DateBucket = ChartDateBucketQuarter
	settings.Sort = ChartSortValueAsc
	data = BuildChartData(chartTestColumns(), rows, settings, location)
	assertChartStrings(t, "quarters by value", chartKeys(data), []string{"2026-Q2", "2026-Q1", "(empty)"})

	created := &TableColumn{BaseInstanceField: &BaseInstanceField{ID: "20261102000000-ccreate", Type: KeyTypeCreated}}
	createdRows := []*TableRow{chartTestRow(&Value{KeyID: created.ID, Type: KeyTypeCreated,
		Created: &ValueCreated{Content: time.Date(2026, 6, 30, 0, 0, 0, 0, location).UnixMilli(), IsNotEmpty: true}})}
	settings = DefaultChartSettings()
	settings.XKeyID, settings.DateBucket = created.ID, ChartDateBucketYear
	if data = BuildChartData([]*TableColumn{created}, createdRows, settings, location); 1 != len(data.Categories) || data.Categories[0].Key != "2026" {
		t.Fatalf("created fields must be bucketed: %+v", data.Categories)
	}
}

func TestChartMultiValueGroupsStacksAndSorts(t *testing.T) {
	relation := func(ids ...string) *Value {
		value := &Value{KeyID: chartTestRelationKey, Type: KeyTypeRelation, Relation: &ValueRelation{BlockIDs: ids}}
		for _, id := range ids {
			value.Relation.Contents = append(value.Relation.Contents, &Value{BlockID: id, Type: KeyTypeBlock, Block: &ValueBlock{Content: "Project " + id[len(id)-1:]}})
		}
		return value
	}
	rows := []*TableRow{
		chartTestRow(chartTags("red", "blue"), chartSelect("A"), chartCheck(true), relation("20261102000000-proj001", "20261102000000-proj002")),
		chartTestRow(chartTags("blue"), chartSelect("B"), chartCheck(false), relation("20261102000000-proj002")),
		chartTestRow(chartTags("blue", "blue"), chartSelect("A"), relation()),
		chartTestRow(chartTags(), chartCheck(true)),
		chartTestRow(chartTags("red")),
	}
	settings := DefaultChartSettings()
	settings.XKeyID = chartTestTagsKey
	data := BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "multi-select", chartKeys(data), []string{"red", "blue"})
	assertChartStrings(t, "multi-select counts", chartValues(data.Series[0].Values), []string{"2", "3"})

	settings.XKeyID = chartTestRelationKey
	settings.ShowEmptyGroups = true
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "relation", chartKeys(data), []string{"20261102000000-proj001", "20261102000000-proj002", "(empty)"})
	assertChartStrings(t, "relation counts", chartValues(data.Series[0].Values), []string{"1", "2", "3"})
	if data.Categories[1].Label != "Project 2" {
		t.Fatalf("relation categories must use the related item title: %+v", data.Categories[1])
	}

	settings = DefaultChartSettings()
	settings.XKeyID = chartTestTagsKey
	settings.StackKeyID = chartTestSelectKey
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	if data.StackType != KeyTypeSelect || 3 != len(data.Series) || data.Series[0].Key != "B" || data.Series[1].Key != "A" ||
		!data.Series[2].Empty || data.Series[0].Color != "2" {
		t.Fatalf("stack series must follow the option order with the empty value last: %+v", data.Series)
	}
	assertChartStrings(t, "stack B", chartValues(data.Series[0].Values), []string{"0", "1"})
	assertChartStrings(t, "stack A", chartValues(data.Series[1].Values), []string{"1", "2"})
	assertChartStrings(t, "stack empty", chartValues(data.Series[2].Values), []string{"1", "0"})
	if data.Series[1].Counts[1] != 2 || 2 != len(data.Series[1].Items[1]) || 0 != len(data.Series[0].Items[0]) {
		t.Fatal("stack points must count and sample their own items")
	}
	settings.ChartType = ChartTypeDonut
	if data = BuildChartData(chartTestColumns(), rows, settings, time.UTC); 1 != len(data.Series) || "" != data.StackType {
		t.Fatal("donut charts must ignore the stack field")
	}

	settings = DefaultChartSettings()
	settings.XKeyID = chartTestSelectKey
	settings.StackKeyID = chartTestCheckKey
	settings.ShowEmptyGroups = true
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "select groups", chartKeys(data), []string{"B", "A", "(empty)"})
	if 2 != len(data.Series) || data.Series[0].Key != "checked" || data.Series[1].Key != "unchecked" {
		t.Fatalf("checkbox stacks must list checked before unchecked: %+v", data.Series)
	}
	assertChartStrings(t, "checked", chartValues(data.Series[0].Values), []string{"0", "1", "1"})
	assertChartStrings(t, "unchecked", chartValues(data.Series[1].Values), []string{"1", "1", "1"})

	settings.StackKeyID = ""
	settings.Sort = ChartSortValueDesc
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "value desc keeps empty last", chartKeys(data), []string{"A", "B", "(empty)"})
	settings.Sort = ChartSortLabelAsc
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "label asc", chartKeys(data), []string{"A", "B", "(empty)"})
	settings.XKeyID = chartTestCheckKey
	settings.Sort = ChartSortDefault
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "checkbox", chartKeys(data), []string{"checked", "unchecked"})
	assertChartStrings(t, "checkbox counts", chartValues(data.Series[0].Values), []string{"2", "3"})
}

func TestChartValueSourcesAndPublishFiltering(t *testing.T) {
	rollup := func(numbers ...float64) *Value {
		value := &Value{KeyID: chartTestRollupKey, Type: KeyTypeRollup, Rollup: &ValueRollup{}}
		for _, number := range numbers {
			value.Rollup.Contents = append(value.Rollup.Contents, &Value{Type: KeyTypeNumber, Number: &ValueNumber{Content: number, IsNotEmpty: true}})
		}
		return value
	}
	template := func(content string) *Value {
		return &Value{KeyID: chartTestTemplateKey, Type: KeyTypeTemplate, Template: &ValueTemplate{Content: content}}
	}
	rows := []*TableRow{
		chartTestRow(chartSelect("A"), rollup(1, 2), template("4")),
		chartTestRow(chartSelect("A"), rollup(), template("not a number")),
		chartTestRow(chartSelect("B"), rollup(5), template(" 1.5 ")),
	}
	settings := DefaultChartSettings()
	settings.XKeyID = chartTestSelectKey
	settings.Measure = ChartMeasureSum
	settings.ValueKeyID = chartTestRollupKey
	data := BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "rollup sums", chartValues(data.Series[0].Values), []string{"5", "3"})
	settings.ValueKeyID = chartTestTemplateKey
	settings.Measure = ChartMeasureAverage
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	assertChartStrings(t, "template averages", chartValues(data.Series[0].Values), []string{"1.5", "4"})

	chart := &Chart{Table: &Table{BaseInstance: &BaseInstance{ID: "20261102000000-cview01"}, Columns: chartTestColumns(),
		Rows: rows, Chart: &settings}}
	ResolveChart(chart)
	if 0 != len(chart.Rows) || 3 != len(chart.ChartRows) || 3 != chart.RowCount || nil == chart.ChartData || 2 != len(chart.ChartData.Categories) {
		t.Fatalf("resolving a chart must keep the rows internally and return only the summary: %+v", chart.ChartData)
	}
	RetainChartRows(chart.Table, func(row *TableRow) bool { return row.ID != rows[2].ID })
	if 2 != chart.RowCount || 1 != len(chart.ChartData.Categories) || chart.ChartData.Categories[0].Key != "A" || 2 != chart.ChartData.ItemCount {
		t.Fatalf("publish filtering must recompute the summary: %+v", chart.ChartData)
	}

	many := make([]*TableRow, 0, 3*ChartItemSampleLimit)
	for i := 0; i < 3*ChartItemSampleLimit; i++ {
		many = append(many, chartTestRow(chartSelect("A")))
	}
	settings = DefaultChartSettings()
	settings.XKeyID = chartTestSelectKey
	data = BuildChartData(chartTestColumns(), many, settings, time.UTC)
	if ChartItemSampleLimit != len(data.Series[0].Items[0]) || 3*ChartItemSampleLimit != data.Series[0].Counts[0] {
		t.Fatal("sample items must be capped per data point")
	}
}

func TestChartLayoutRegistration(t *testing.T) {
	chart := &Chart{Table: &Table{}}
	if chart.GetType() != LayoutTypeChart || TableFromViewable(chart) != chart.Table {
		t.Fatal("chart must expose its table")
	}
	view := &View{ID: "20261102000000-cview02", LayoutType: LayoutTypeChart}
	database := &AttributeView{Views: []*View{view}}
	if err := database.ValidateLayouts(); nil == err {
		t.Fatal("missing chart layout must fail validation")
	}
	view.Chart = &LayoutChart{LayoutTable: NewLayoutTable(), Settings: ChartSettings{ChartType: "pie"}}
	if err := database.ValidateLayouts(); nil == err {
		t.Fatal("invalid chart type must fail validation")
	}
	view.Chart.Settings = DefaultChartSettings()
	if err := database.ValidateLayouts(); nil != err {
		t.Fatalf("valid chart rejected: %v", err)
	}
	if view.GetTableLayout() != view.Chart.LayoutTable || view.GetBaseLayout() != view.Chart.BaseLayout ||
		view.TableLayouts()[len(view.TableLayouts())-1] != view.Chart.LayoutTable {
		t.Fatal("chart must share the row and column layout helpers")
	}
	view.Group = &ViewGroup{Field: chartTestStatusKey}
	if view.IsGroupView() || !KeepsPrimaryKeyVisible(LayoutTypeChart) {
		t.Fatal("chart must not group items and must keep its primary key visible")
	}
	keyIDMap := map[string]string{chartTestStatusKey: "20261102000000-cnewsta", chartTestNumberKey: "20261102000000-cnewnum"}
	settings := ChartSettings{XKeyID: chartTestStatusKey, ValueKeyID: chartTestNumberKey, StackKeyID: chartTestSelectKey}
	settings.remapKeyIDs(keyIDMap)
	if settings.XKeyID != "20261102000000-cnewsta" || settings.ValueKeyID != "20261102000000-cnewnum" || settings.StackKeyID != chartTestSelectKey {
		t.Fatalf("chart field bindings must follow cloned keys: %+v", settings)
	}
}

func TestChartFieldTypes(t *testing.T) {
	for _, typ := range KeyTypes() {
		group := typ != KeyTypeMAsset && typ != KeyTypeRollup && typ != KeyTypeLineNumber &&
			typ != KeyTypeLocation
		value := typ == KeyTypeNumber || typ == KeyTypeRollup || typ == KeyTypeTemplate
		stack := typ == KeyTypeSelect || typ == KeyTypeStatus || typ == KeyTypeCheckbox || typ == KeyTypeBlock || typ == KeyTypeRelation
		if IsChartGroupKeyType(typ) != group || IsChartValueKeyType(typ) != value || IsChartStackKeyType(typ) != stack {
			t.Fatalf("chart field capabilities changed for %s", typ)
		}
	}
}

func TestChartDecimalSum(t *testing.T) {
	var sum chartSum
	for _, value := range []float64{0.1, 0.2, -0.05, 1e-7} {
		sum.add(value)
	}
	if got := sum.float64(); got != 0.2500001 || nil != sum.fallback {
		t.Fatalf("short decimals must be summed exactly: %v", got)
	}
	if got := sum.average(5); got != 0.05000002 {
		t.Fatalf("decimal average: %v", got)
	}
	sum.add(9e18)
	sum.add(9e18)
	if nil == sum.fallback || sum.float64() != 1.8e19+0.2500001 {
		t.Fatalf("overflowing sums must fall back to exact rational arithmetic: %v", sum.float64())
	}
	if _, _, ok := chartDecimal(1e300); ok {
		t.Fatal("numbers beyond int64 must use the fallback")
	}
	if mantissa, scale, ok := chartDecimal(-12.375); !ok || mantissa != -12375 || scale != 3 {
		t.Fatalf("decimal split: %d %d %t", mantissa, scale, ok)
	}
}

func TestChartGroupByPrimaryKey(t *testing.T) {
	rows := []*TableRow{chartTestRow(chartNumber(3)), chartTestRow(chartNumber(5)), chartTestRow()}
	rows[1].Cells[0].Value.Block.Content = rows[0].Cells[0].Value.Block.Content // 同名条目仍分开显示
	settings := DefaultChartSettings()
	settings.XKeyID = chartTestBlockKey
	settings.Measure = ChartMeasureSum
	settings.ValueKeyID = chartTestNumberKey
	data := BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	if 3 != len(data.Categories) {
		t.Fatalf("expected one category per item, got %d", len(data.Categories))
	}
	for i, category := range data.Categories {
		if "" == category.Label {
			t.Fatalf("category %d has no label", i)
		}
	}
	total := 0.0
	for _, value := range data.Series[0].Values {
		if nil != value {
			total += *value
		}
	}
	if 8 != total {
		t.Fatalf("sum per item = %v, want 8", total)
	}
}

// mods: stacking by a relation (one part per related page) or by the page name (one part per page).
func TestChartStacksByRelationAndName(t *testing.T) {
	relation := func(ids ...string) *Value {
		value := &Value{KeyID: chartTestRelationKey, Type: KeyTypeRelation, Relation: &ValueRelation{BlockIDs: ids}}
		for _, id := range ids {
			value.Relation.Contents = append(value.Relation.Contents, &Value{BlockID: id, Type: KeyTypeBlock, Block: &ValueBlock{Content: "Project " + id[len(id)-1:]}})
		}
		return value
	}
	p1, p2 := "20261102000000-proj001", "20261102000000-proj002"
	rows := []*TableRow{
		chartTestRow(chartDate(2026, time.November, 2), chartNumber(1), relation(p2)),
		chartTestRow(chartDate(2026, time.November, 2), chartNumber(0.5), relation(p1)),
		chartTestRow(chartDate(2026, time.November, 2), chartNumber(2), relation(p1, p2)),
		chartTestRow(chartDate(2026, time.November, 3), chartNumber(1), relation()),
	}
	settings := DefaultChartSettings()
	settings.XKeyID, settings.DateBucket = chartTestDateKey, ChartDateBucketDay
	settings.Measure, settings.ValueKeyID = ChartMeasureSum, chartTestNumberKey
	settings.StackKeyID = chartTestRelationKey
	data := BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	if data.StackType != KeyTypeRelation {
		t.Fatalf("relation fields must be accepted as stacks: %q", data.StackType)
	}
	assertChartStrings(t, "days", chartKeys(data), []string{"2026-11-02", "2026-11-03"})
	totals := []string{}
	for _, category := range data.Categories {
		totals = append(totals, fmt.Sprint(*category.Value))
	}
	assertChartStrings(t, "day totals count each item once", totals, []string{"3.5", "1"})
	if 3 != len(data.Series) || data.Series[0].Label != "Project 1" || data.Series[1].Label != "Project 2" || !data.Series[2].Empty {
		t.Fatalf("relation stacks must be sorted by title with the empty value last: %+v", data.Series)
	}
	assertChartStrings(t, "project 1", chartValues(data.Series[0].Values), []string{"2.5", "0"})
	assertChartStrings(t, "project 2", chartValues(data.Series[1].Values), []string{"3", "0"})
	assertChartStrings(t, "no project", chartValues(data.Series[2].Values), []string{"0", "1"})

	settings.StackKeyID = chartTestBlockKey
	data = BuildChartData(chartTestColumns(), rows, settings, time.UTC)
	if data.StackType != KeyTypeBlock || 4 != len(data.Series) {
		t.Fatalf("the page name must be accepted as a stack, one part per page: %+v", data.Series)
	}
	for i := 1; i < len(data.Series); i++ {
		if data.Series[i-1].Label > data.Series[i].Label {
			t.Fatalf("name stacks must be sorted by title: %+v", data.Series)
		}
	}
	if !IsChartStackKeyType(KeyTypeRelation) || !IsChartStackKeyType(KeyTypeBlock) || IsChartStackKeyType(KeyTypeMSelect) {
		t.Fatal("chart stack capabilities: relation and block allowed, multi-select not")
	}
}

func TestChartCumulative(t *testing.T) {
	location := time.UTC
	rows := []*TableRow{
		chartTestRow(chartDate(2026, 1, 5), chartNumber(100), chartSelect("A")),
		chartTestRow(chartDate(2026, 1, 20), chartNumber(-30.5), chartSelect("B")),
		chartTestRow(chartDate(2026, 3, 2), chartNumber(-20), chartSelect("A")),
		chartTestRow(chartNumber(7), chartSelect("A")),
	}
	settings := DefaultChartSettings()
	settings.ChartType = ChartTypeLine
	settings.XKeyID, settings.Measure, settings.ValueKeyID = chartTestDateKey, ChartMeasureSum, chartTestNumberKey
	settings.Cumulative, settings.ShowEmptyGroups = true, true
	data := BuildChartData(chartTestColumns(), rows, settings, location)
	assertChartStrings(t, "keys", chartKeys(data), []string{"2026-01", "2026-02", "2026-03", "(empty)"})
	assertChartStrings(t, "running totals", chartValues(data.Series[0].Values), []string{"69.5", "69.5", "49.5", "7"})
	if nil == data.Categories[2].Value || 49.5 != *data.Categories[2].Value || nil == data.Total || 56.5 != *data.Total {
		t.Fatalf("category running total and overall total: %+v %v", data.Categories[2].Value, data.Total)
	}

	settings.StackKeyID, settings.ShowEmptyGroups = chartTestSelectKey, false
	data = BuildChartData(chartTestColumns(), rows, settings, location)
	assertChartStrings(t, "stack keys", chartKeys(data), []string{"2026-01", "2026-03"})
	for _, series := range data.Series {
		switch series.Key {
		case "A":
			assertChartStrings(t, "A running", chartValues(series.Values), []string{"100", "80"})
		case "B":
			assertChartStrings(t, "B running", chartValues(series.Values), []string{"-30.5", "-30.5"})
		}
	}

	settings.StackKeyID = ""
	settings.ChartType = ChartTypeDonut
	data = BuildChartData(chartTestColumns(), rows, settings, location)
	assertChartStrings(t, "donut ignores cumulative", chartValues(data.Series[0].Values), []string{"69.5", "-20"})
	settings.ChartType, settings.Measure = ChartTypeBar, ChartMeasureAverage
	data = BuildChartData(chartTestColumns(), rows, settings, location)
	assertChartStrings(t, "average ignores cumulative", chartValues(data.Series[0].Values), []string{"34.75", "-20"})
}
