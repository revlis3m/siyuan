// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"reflect"
	"testing"
	"time"

	"github.com/88250/lute/ast"
	"github.com/siyuan-note/siyuan/kernel/av"
	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/sql"
	"github.com/siyuan-note/siyuan/kernel/util"
)

func setupAttributeViewChartTest(t *testing.T) (*av.AttributeView, *av.View) {
	setupAttributeViewValidationTest(t)
	setAttributeViewListTestLangs()
	util.AttrViewLangs["en"]["calendar"] = "Calendar"
	util.AttrViewLangs["en"]["timeline"] = "Timeline"
	util.AttrViewLangs["en"]["chart"] = "Chart"
	attrView := newAttributeViewWithLayout(ast.NewNodeID(), av.LayoutTypeChart)
	return attrView, attrView.Views[0]
}

func newAttributeViewChartStatusKey() *av.Key {
	status := av.NewKey(ast.NewNodeID(), "Status", "", av.KeyTypeStatus)
	status.Options = []*av.SelectOption{
		{Name: "Done", Color: "11", Group: av.StatusGroupComplete},
		{Name: "Todo", Color: "5", Group: av.StatusGroupTodo},
		{Name: "Doing", Color: "6", Group: av.StatusGroupInProgress},
	}
	return status
}

func TestAttributeViewChartDefaultsAndLayoutSwitch(t *testing.T) {
	attrView, view := setupAttributeViewChartTest(t)
	selectKey := attrView.KeyValues[1].Key
	if view.LayoutType != av.LayoutTypeChart || nil == view.Chart || view.Name != "Chart" {
		t.Fatalf("expected a chart view: %+v", view)
	}
	settings := view.Chart.Settings
	if settings.XKeyID != selectKey.ID || settings.ChartType != av.ChartTypeBar || settings.Measure != av.ChartMeasureCount ||
		settings.Height != av.ChartDefaultHeight || !settings.ShowLegend {
		t.Fatalf("a new chart must group by its select field: %+v", settings)
	}
	assertListDefaultFields(t, attrView, &av.View{LayoutType: av.LayoutTypeList, List: view.Chart.LayoutTable})

	date := av.NewKey(ast.NewNodeID(), "Date", "", av.KeyTypeDate)
	addAttributeViewKey(attrView, nil, date, "")
	status := newAttributeViewChartStatusKey()
	addAttributeViewKey(attrView, view, status, "")
	if field := getAttributeViewField(view, date.ID); nil == field || !field.Hidden {
		t.Fatal("a field added elsewhere must stay hidden in the chart")
	}
	defaults := newAttributeViewChartLayout(attrView, attributeViewFieldIDs(view)).Settings
	if defaults.XKeyID != status.ID || defaults.DateBucket != av.ChartDateBucketMonth {
		t.Fatalf("chart defaults must prefer the status field: %+v", defaults)
	}
	if err := setAttributeViewFieldsHidden(attrView, attrView.GetBlockKeyValues().Key.ID, []string{view.ID}, true); nil == err {
		t.Fatal("chart primary field must remain visible")
	}

	view.Chart.Settings = defaults
	view.Chart.Settings.ChartType = av.ChartTypeDonut
	configured := view.Chart.Settings
	view.Group = &av.ViewGroup{Field: selectKey.ID}
	if view.IsGroupView() {
		t.Fatal("chart must not group items")
	}
	chart := view.Chart
	for _, layout := range []av.LayoutType{av.LayoutTypeTable, av.LayoutTypeList, av.LayoutTypeCalendar, av.LayoutTypeTimeline, av.LayoutTypeGallery, av.LayoutTypeKanban} {
		if err := changeAttrViewLayout(attrView, view, layout); nil != err {
			t.Fatal(err)
		}
		grouped := av.LayoutTypeCalendar != layout && av.LayoutTypeTimeline != layout
		if view.LayoutType != layout || view.IsGroupView() != grouped {
			t.Fatalf("switching to %s must restore grouping", layout)
		}
		if err := changeAttrViewLayout(attrView, view, av.LayoutTypeChart); nil != err {
			t.Fatal(err)
		}
		if view.Chart != chart || view.Chart.Settings != configured || view.Name != "Chart" || view.IsGroupView() {
			t.Fatalf("chart settings were lost after switching to %s: %+v", layout, view.Chart.Settings)
		}
	}

	table := av.NewTableView()
	attrView.Views = append(attrView.Views, table)
	if err := changeAttrViewLayout(attrView, table, av.LayoutTypeChart); nil != err {
		t.Fatal(err)
	}
	if table.Name != "Chart" || nil == table.Chart || table.Chart.Settings.XKeyID != status.ID {
		t.Fatalf("switching a table to a chart must create default settings: %+v", table.Chart)
	}
	if err := changeAttrViewLayout(attrView, table, av.LayoutTypeTable); nil != err || table.Name != "Table" {
		t.Fatalf("switching back must restore the default name: %v %s", err, table.Name)
	}

	cloned := av.NewChartView()
	if err := cloneAttributeViewLayouts(cloned, view); nil != err {
		t.Fatal(err)
	}
	cloned.Chart.Columns[0].Hidden = true
	cloned.Chart.Settings.ChartType = av.ChartTypeLine
	if view.Chart.Columns[0].Hidden || view.Chart.ID == cloned.Chart.ID || view.Chart.Settings.ChartType != av.ChartTypeDonut {
		t.Fatal("chart clone must be independent")
	}
	copied := attrView.Clone()
	if nil == copied {
		t.Fatal("database clone failed")
	}
	copiedStatus, err := copied.GetKey(copied.Views[0].Chart.Settings.XKeyID)
	if nil != err || copiedStatus.Type != av.KeyTypeStatus || copiedStatus.ID == status.ID {
		t.Fatal("database clone must bind its own chart fields")
	}

	if err = av.SaveAttributeView(attrView); nil != err {
		t.Fatal(err)
	}
	cache.ClearAVCache()
	stored, err := av.ParseAttributeView(attrView.ID)
	if nil != err || !reflect.DeepEqual(stored.Views[0].Chart, view.Chart) {
		t.Fatalf("chart roundtrip: %v", err)
	}
}

func TestAttributeViewChartRenderAppliesFiltersAndAggregates(t *testing.T) {
	attrView, view := setupAttributeViewChartTest(t)
	status := newAttributeViewChartStatusKey()
	addAttributeViewKey(attrView, view, status, "")
	number := av.NewKey(ast.NewNodeID(), "Points", "", av.KeyTypeNumber)
	addAttributeViewKey(attrView, view, number, status.ID)
	date := av.NewKey(ast.NewNodeID(), "Due", "", av.KeyTypeDate)
	addAttributeViewKey(attrView, view, date, number.ID)
	text := av.NewKey(ast.NewNodeID(), "Note", "", av.KeyTypeText)
	addAttributeViewKey(attrView, view, text, date.ID)
	view.Chart.Settings = av.DefaultChartSettings()
	view.Chart.Settings.XKeyID = status.ID
	view.PageSize = 1
	view.Sorts = []*av.ViewSort{{Column: number.ID, Order: av.SortOrderDesc}}

	primary := attrView.GetBlockKeyValues()
	statusValues, _ := attrView.GetKeyValues(status.ID)
	numberValues, _ := attrView.GetKeyValues(number.ID)
	dateValues, _ := attrView.GetKeyValues(date.ID)
	textValues, _ := attrView.GetKeyValues(text.ID)
	type fixture struct {
		note, status string
		points       float64
		due          time.Time
	}
	fixtures := []fixture{
		{"a", "Done", 3, time.Date(2026, 1, 5, 9, 0, 0, 0, time.Local)},
		{"b", "Done", 5, time.Date(2026, 1, 20, 9, 0, 0, 0, time.Local)},
		{"c", "Doing", 1.5, time.Date(2026, 3, 2, 9, 0, 0, 0, time.Local)},
		{"d", "", 4, time.Date(2026, 3, 30, 9, 0, 0, 0, time.Local)},
		{"", "Todo", 100, time.Date(2026, 5, 1, 9, 0, 0, 0, time.Local)},
	}
	for _, item := range fixtures {
		id := ast.NewNodeID()
		view.ItemIDs = append(view.ItemIDs, id)
		primary.Values = append(primary.Values, &av.Value{ID: ast.NewNodeID(), KeyID: primary.Key.ID, BlockID: id,
			Type: av.KeyTypeBlock, IsDetached: true, Block: &av.ValueBlock{Content: "Item " + item.note}})
		if "" != item.status {
			statusValues.Values = append(statusValues.Values, &av.Value{ID: ast.NewNodeID(), KeyID: status.ID, BlockID: id,
				Type: av.KeyTypeStatus, MSelect: []*av.ValueSelect{{Content: item.status}}})
		}
		numberValues.Values = append(numberValues.Values, &av.Value{ID: ast.NewNodeID(), KeyID: number.ID, BlockID: id,
			Type: av.KeyTypeNumber, Number: &av.ValueNumber{Content: item.points, IsNotEmpty: true}})
		dateValues.Values = append(dateValues.Values, &av.Value{ID: ast.NewNodeID(), KeyID: date.ID, BlockID: id,
			Type: av.KeyTypeDate, Date: &av.ValueDate{Content: item.due.UnixMilli(), IsNotEmpty: true}})
		textValues.Values = append(textValues.Values, &av.Value{ID: ast.NewNodeID(), KeyID: text.ID, BlockID: id,
			Type: av.KeyTypeText, Text: &av.ValueText{Content: item.note}})
	}
	view.Filters = []*av.ViewFilter{{Column: text.ID, Operator: av.FilterOperatorIsNotEmpty, Value: &av.Value{Type: av.KeyTypeText}}}

	render := func() *av.Chart {
		t.Helper()
		chart, ok := sql.RenderView(attrView, view, "", false).(*av.Chart)
		if !ok {
			t.Fatal("chart view must render a chart instance")
		}
		if _, _, err := renderViewableInstance(chart, view, attrView, 1, 1, false, "", sql.NewAttributeViewRenderContext()); nil != err {
			t.Fatal(err)
		}
		if nil == chart.ChartData || 0 != len(chart.Rows) {
			t.Fatal("chart must return the summary without rows")
		}
		return chart
	}
	keys := func(data *av.ChartData) (ret []string) {
		for _, category := range data.Categories {
			if category.Empty {
				ret = append(ret, "(empty)")
			} else {
				ret = append(ret, category.Key)
			}
		}
		return
	}
	values := func(series *av.ChartSeries) (ret []float64) {
		for _, value := range series.Values {
			if nil == value {
				ret = append(ret, -1)
			} else {
				ret = append(ret, *value)
			}
		}
		return
	}

	chart := render()
	if chart.RowCount != 4 || chart.ChartData.ItemCount != 4 || len(chart.ChartRows) != 4 {
		t.Fatalf("filters must apply and pagination must not: %d %d", chart.RowCount, chart.ChartData.ItemCount)
	}
	if got := keys(chart.ChartData); !reflect.DeepEqual(got, []string{"Doing", "Done"}) {
		t.Fatalf("status groups must follow the status order: %v", got)
	}
	if got := values(chart.ChartData.Series[0]); !reflect.DeepEqual(got, []float64{1, 2}) {
		t.Fatalf("unexpected counts: %v", got)
	}

	view.Chart.Settings.Measure = av.ChartMeasureSum
	view.Chart.Settings.ValueKeyID = number.ID
	view.Chart.Settings.ShowEmptyGroups = true
	chart = render()
	if got := keys(chart.ChartData); !reflect.DeepEqual(got, []string{"Todo", "Doing", "Done", "(empty)"}) {
		t.Fatalf("empty groups must list every status option and the empty value: %v", got)
	}
	if got := values(chart.ChartData.Series[0]); !reflect.DeepEqual(got, []float64{0, 1.5, 8, 4}) {
		t.Fatalf("unexpected sums: %v", got)
	}
	if nil == chart.ChartData.Total || 13.5 != *chart.ChartData.Total {
		t.Fatal("the total must exclude filtered items")
	}

	view.Chart.Settings = av.DefaultChartSettings()
	view.Chart.Settings.XKeyID = date.ID
	view.Chart.Settings.DateBucket = av.ChartDateBucketQuarter
	view.Chart.Settings.StackKeyID = status.ID
	chart = render()
	if got := keys(chart.ChartData); !reflect.DeepEqual(got, []string{"2026-Q1"}) {
		t.Fatalf("dates must be grouped by quarter: %v", got)
	}
	if 3 != len(chart.ChartData.Series) || chart.ChartData.Series[0].Key != "Doing" || chart.ChartData.Series[1].Key != "Done" ||
		!chart.ChartData.Series[2].Empty || 2 != *chart.ChartData.Series[1].Values[0] {
		t.Fatalf("stack series must follow the status order: %+v", chart.ChartData.Series)
	}

	view.Chart.Settings.XKeyID = text.ID
	view.Chart.Settings.StackKeyID = ""
	view.Chart.Settings.Sort = av.ChartSortLabelAsc
	chart = render()
	if got := keys(chart.ChartData); !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Fatalf("text groups must sort by label: %v", got)
	}

	template := av.NewKey(ast.NewNodeID(), "Score", "", av.KeyTypeTemplate)
	template.Template = "1"
	addAttributeViewKey(attrView, view, template, text.ID)
	view.Chart.Settings.ValueKeyID = ""
	if !shouldDeferAttributeViewTemplateValues(attrView, view, "", false) {
		t.Fatal("unbound template fields may be computed after the chart summary")
	}
	view.Chart.Settings.ValueKeyID = template.ID
	if shouldDeferAttributeViewTemplateValues(attrView, view, "", false) {
		t.Fatal("a template field bound to the chart must be computed before the summary")
	}
}

func TestAttributeViewChartSettingsTransaction(t *testing.T) {
	attrView, view := setupAttributeViewChartTest(t)
	number := av.NewKey(ast.NewNodeID(), "Points", "", av.KeyTypeNumber)
	addAttributeViewKey(attrView, view, number, "")
	if err := av.SaveAttributeView(attrView); nil != err {
		t.Fatal(err)
	}
	operation := func(viewID string, data any) *Operation {
		return &Operation{Action: "setAttrViewChart", AvID: attrView.ID, ViewID: viewID, Data: data}
	}
	selectKey := attrView.KeyValues[1].Key
	next := map[string]any{"chartType": "horizontalBar", "xKeyID": selectKey.ID, "dateBucket": "week", "measure": "average",
		"valueKeyID": number.ID, "stackKeyID": "", "sort": "valueDesc", "showEmptyGroups": true, "showLegend": false,
		"showDataLabels": true, "height": 480}
	if err := setAttrViewChart(operation(view.ID, next)); nil != err {
		t.Fatal(err)
	}
	cache.ClearAVCache()
	stored, err := av.ParseAttributeView(attrView.ID)
	if nil != err {
		t.Fatal(err)
	}
	want := av.ChartSettings{ChartType: av.ChartTypeHorizontalBar, XKeyID: selectKey.ID, DateBucket: av.ChartDateBucketWeek,
		Measure: av.ChartMeasureAverage, ValueKeyID: number.ID, Sort: av.ChartSortValueDesc, ShowEmptyGroups: true,
		ShowDataLabels: true, Height: 480}
	if stored.Views[0].Chart.Settings != want {
		t.Fatalf("chart settings were not saved: %+v", stored.Views[0].Chart.Settings)
	}
	for _, invalid := range []any{nil, map[string]any{"chartType": "pie"}, map[string]any{"height": 20},
		map[string]any{"xKeyID": "../key"}, map[string]any{"measure": "median"}} {
		if err = setAttrViewChart(operation(view.ID, invalid)); nil == err {
			t.Fatalf("invalid chart settings accepted: %v", invalid)
		}
	}
	table := av.NewTableView()
	stored.Views = append(stored.Views, table)
	if err = av.SaveAttributeView(stored); nil != err {
		t.Fatal(err)
	}
	if err = setAttrViewChart(operation(table.ID, next)); nil == err {
		t.Fatal("chart settings must reject other layouts")
	}
}
