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

func setupAttributeViewTimelineTest(t *testing.T) (*av.AttributeView, *av.View) {
	setupAttributeViewValidationTest(t)
	setAttributeViewListTestLangs()
	util.AttrViewLangs["en"]["calendar"] = "Calendar"
	util.AttrViewLangs["en"]["timeline"] = "Timeline"
	attrView := newAttributeViewWithLayout(ast.NewNodeID(), av.LayoutTypeTimeline)
	return attrView, attrView.Views[0]
}

func TestAttributeViewTimelineDefaultsAndLayoutSwitch(t *testing.T) {
	attrView, view := setupAttributeViewTimelineTest(t)
	selectKey := attrView.KeyValues[1].Key
	if view.LayoutType != av.LayoutTypeTimeline || nil == view.Timeline || view.Name != "Timeline" {
		t.Fatalf("expected a timeline view: %+v", view)
	}
	settings := view.Timeline.Settings
	if settings.StartKeyID != "" || settings.Zoom != av.TimelineZoomWeek || settings.ColorBy != selectKey.ID {
		t.Fatalf("unexpected default timeline settings without a date field: %+v", settings)
	}
	assertListDefaultFields(t, attrView, &av.View{LayoutType: av.LayoutTypeList, List: view.Timeline.LayoutTable})

	date := av.NewKey(ast.NewNodeID(), "Date", "", av.KeyTypeDate)
	addAttributeViewKey(attrView, view, date, "")
	status := av.NewKey(ast.NewNodeID(), "Status", "", av.KeyTypeStatus)
	addAttributeViewKey(attrView, nil, status, "")
	if field := getAttributeViewField(view, date.ID); nil == field || field.Hidden {
		t.Fatal("a field added from the timeline must be shown in its property column")
	}
	if field := getAttributeViewField(view, status.ID); nil == field || !field.Hidden {
		t.Fatal("a field added elsewhere must stay hidden in the timeline")
	}
	defaults := newAttributeViewTimelineLayout(attrView, attributeViewFieldIDs(view)).Settings
	if defaults.StartKeyID != date.ID || defaults.EndKeyID != "" || defaults.ColorBy != status.ID || defaults.Zoom != av.TimelineZoomWeek {
		t.Fatalf("timeline defaults must use the first date and status fields: %+v", defaults)
	}

	view.Timeline.Settings = defaults
	view.Group = &av.ViewGroup{Field: selectKey.ID}
	if view.IsGroupView() {
		t.Fatal("timeline must not group items")
	}
	timeline := view.Timeline
	for _, layout := range []av.LayoutType{av.LayoutTypeTable, av.LayoutTypeList, av.LayoutTypeCalendar, av.LayoutTypeGallery, av.LayoutTypeKanban} {
		if err := changeAttrViewLayout(attrView, view, layout); nil != err {
			t.Fatal(err)
		}
		if view.LayoutType != layout || view.IsGroupView() != (av.LayoutTypeCalendar != layout) {
			t.Fatalf("switching to %s must restore grouping", layout)
		}
		if err := changeAttrViewLayout(attrView, view, av.LayoutTypeTimeline); nil != err {
			t.Fatal(err)
		}
		if view.Timeline != timeline || view.Timeline.Settings != defaults || view.Name != "Timeline" {
			t.Fatalf("timeline settings were lost after switching to %s: %+v", layout, view.Timeline.Settings)
		}
	}

	table := av.NewTableView()
	attrView.Views = append(attrView.Views, table)
	if err := changeAttrViewLayout(attrView, table, av.LayoutTypeTimeline); nil != err {
		t.Fatal(err)
	}
	if table.Name != "Timeline" || nil == table.Timeline || table.Timeline.Settings.StartKeyID != date.ID {
		t.Fatalf("switching a table to timeline must create default settings: %+v", table.Timeline)
	}
	if err := changeAttrViewLayout(attrView, table, av.LayoutTypeTable); nil != err || table.Name != "Table" {
		t.Fatalf("switching back must restore the default name: %v %s", err, table.Name)
	}

	cloned := av.NewTimelineView()
	if err := cloneAttributeViewLayouts(cloned, view); nil != err {
		t.Fatal(err)
	}
	cloned.Timeline.Columns[0].Hidden = true
	cloned.Timeline.Settings.Zoom = av.TimelineZoomDay
	if view.Timeline.Columns[0].Hidden || view.Timeline.ID == cloned.Timeline.ID || view.Timeline.Settings.Zoom != av.TimelineZoomWeek {
		t.Fatal("timeline clone must be independent")
	}
	copied := attrView.Clone()
	if nil == copied {
		t.Fatal("database clone failed")
	}
	copiedSettings := copied.Views[0].Timeline.Settings
	copiedDate, dateErr := copied.GetKey(copiedSettings.StartKeyID)
	copiedStatus, statusErr := copied.GetKey(copiedSettings.ColorBy)
	if nil != dateErr || nil != statusErr || copiedDate.Type != av.KeyTypeDate || copiedStatus.Type != av.KeyTypeStatus ||
		copiedSettings.StartKeyID == date.ID {
		t.Fatal("database clone must bind its own timeline fields")
	}

	if err := av.SaveAttributeView(attrView); nil != err {
		t.Fatal(err)
	}
	cache.ClearAVCache()
	stored, err := av.ParseAttributeView(attrView.ID)
	if nil != err || !reflect.DeepEqual(stored.Views[0].Timeline, view.Timeline) {
		t.Fatalf("timeline roundtrip: %v", err)
	}
	if err = setAttributeViewFieldsHidden(attrView, attrView.GetBlockKeyValues().Key.ID, []string{view.ID}, true); nil == err {
		t.Fatal("timeline primary field must remain visible")
	}
}

func TestAttributeViewTimelineRenderRespectsFiltersAndResolvesDates(t *testing.T) {
	attrView, view := setupAttributeViewTimelineTest(t)
	start := av.NewKey(ast.NewNodeID(), "Start", "", av.KeyTypeDate)
	addAttributeViewKey(attrView, view, start, "")
	end := av.NewKey(ast.NewNodeID(), "End", "", av.KeyTypeDate)
	addAttributeViewKey(attrView, view, end, start.ID)
	text := av.NewKey(ast.NewNodeID(), "Note", "", av.KeyTypeText)
	addAttributeViewKey(attrView, view, text, end.ID)
	view.Timeline.Settings = av.TimelineSettings{StartKeyID: start.ID, Zoom: av.TimelineZoomMonth}
	view.PageSize = 1

	primary := attrView.GetBlockKeyValues()
	startValues, _ := attrView.GetKeyValues(start.ID)
	endValues, _ := attrView.GetKeyValues(end.ID)
	textValues, _ := attrView.GetKeyValues(text.ID)
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local).UnixMilli()
	day := int64(86400000)
	type fixture struct {
		note       string
		start, end *av.ValueDate
	}
	fixtures := []fixture{
		{"single", &av.ValueDate{Content: base, IsNotEmpty: true, IsNotTime: true}, nil},
		{"range", &av.ValueDate{Content: base + day, IsNotEmpty: true, IsNotTime: true, HasEndDate: true, Content2: base + 4*day, IsNotEmpty2: true}, nil},
		{"end field", &av.ValueDate{Content: base + 2*day, IsNotEmpty: true, IsNotTime: true}, &av.ValueDate{Content: base + 9*day, IsNotEmpty: true, IsNotTime: true}},
		{"unscheduled", &av.ValueDate{}, nil},
		{"", &av.ValueDate{Content: base, IsNotEmpty: true, IsNotTime: true}, nil},
		{"", &av.ValueDate{}, nil},
	}
	ids := make([]string, len(fixtures))
	for i, item := range fixtures {
		id := ast.NewNodeID()
		ids[i] = id
		view.ItemIDs = append(view.ItemIDs, id)
		primary.Values = append(primary.Values, &av.Value{ID: ast.NewNodeID(), KeyID: primary.Key.ID, BlockID: id,
			Type: av.KeyTypeBlock, IsDetached: true, Block: &av.ValueBlock{Content: "Item " + item.note}})
		startValues.Values = append(startValues.Values, &av.Value{ID: ast.NewNodeID(), KeyID: start.ID, BlockID: id,
			Type: av.KeyTypeDate, Date: item.start})
		if nil != item.end {
			endValues.Values = append(endValues.Values, &av.Value{ID: ast.NewNodeID(), KeyID: end.ID, BlockID: id,
				Type: av.KeyTypeDate, Date: item.end})
		}
		textValues.Values = append(textValues.Values, &av.Value{ID: ast.NewNodeID(), KeyID: text.ID, BlockID: id,
			Type: av.KeyTypeText, Text: &av.ValueText{Content: item.note}})
	}
	view.Filters = []*av.ViewFilter{{Column: text.ID, Operator: av.FilterOperatorIsNotEmpty, Value: &av.Value{Type: av.KeyTypeText}}}

	render := func() *av.Timeline {
		t.Helper()
		timeline, ok := sql.RenderView(attrView, view, "", false).(*av.Timeline)
		if !ok {
			t.Fatal("timeline view must render a timeline instance")
		}
		if _, _, err := renderViewableInstance(timeline, view, attrView, 1, 1, false, "", sql.NewAttributeViewRenderContext()); nil != err {
			t.Fatal(err)
		}
		return timeline
	}
	timeline := render()
	if len(timeline.Rows) != 3 || len(timeline.TimelineItems) != 3 || timeline.RowCount != 3 {
		t.Fatalf("filtered scheduled items must not be paginated: rows=%d items=%d", len(timeline.Rows), len(timeline.TimelineItems))
	}
	if len(timeline.TimelineUnscheduled) != 1 || timeline.TimelineUnscheduled[0].ID != ids[3] {
		t.Fatal("only filtered rows without a start date must be unscheduled")
	}
	got := map[string]*av.TimelineItem{}
	for _, item := range timeline.TimelineItems {
		got[item.ID] = item
	}
	if item := got[ids[0]]; nil == item || item.Start != base || item.End != base || !item.IsNotTime || item.Title != "Item single" {
		t.Fatalf("single day item: %+v", item)
	}
	if item := got[ids[1]]; nil == item || item.Start != base+day || item.End != base+4*day {
		t.Fatalf("start date end must close the interval: %+v", item)
	}
	if item := got[ids[2]]; nil == item || item.End != base+2*day {
		t.Fatalf("unbound end field must be ignored: %+v", item)
	}
	if nil != got[ids[4]] {
		t.Fatal("filtered items must not appear on the timeline")
	}

	view.Timeline.Settings.EndKeyID = end.ID
	view.Sorts = []*av.ViewSort{{Column: start.ID, Order: av.SortOrderDesc}}
	timeline = render()
	if item := timeline.TimelineItems[0]; item.ID != ids[2] || item.End != base+9*day {
		t.Fatalf("end field and sorting must apply: %+v", item)
	}
	if item := timeline.TimelineItems[1]; item.ID != ids[1] || item.End != base+4*day {
		t.Fatalf("an empty end field must fall back to the start date end: %+v", item)
	}

	view.Timeline.Settings.StartKeyID = text.ID
	timeline = render()
	if len(timeline.Rows) != 0 || len(timeline.TimelineUnscheduled) != 4 {
		t.Fatal("a non-date start field must list every filtered row as unscheduled")
	}
}

func TestAttributeViewTimelineSettingsAndNewItemDates(t *testing.T) {
	attrView, view := setupAttributeViewTimelineTest(t)
	start := av.NewKey(ast.NewNodeID(), "Start", "", av.KeyTypeDate)
	addAttributeViewKey(attrView, view, start, "")
	end := av.NewKey(ast.NewNodeID(), "End", "", av.KeyTypeDate)
	addAttributeViewKey(attrView, view, end, start.ID)
	view.Timeline.Settings.StartKeyID = start.ID
	if err := av.SaveAttributeView(attrView); nil != err {
		t.Fatal(err)
	}

	operation := func(data any) *Operation {
		return &Operation{Action: "setAttrViewTimeline", AvID: attrView.ID, ViewID: view.ID, Data: data}
	}
	next := map[string]any{"startKeyID": start.ID, "endKeyID": end.ID, "zoom": "quarter", "colorBy": "conditional"}
	if err := setAttrViewTimeline(operation(next)); nil != err {
		t.Fatal(err)
	}
	cache.ClearAVCache()
	stored, err := av.ParseAttributeView(attrView.ID)
	if nil != err {
		t.Fatal(err)
	}
	want := av.TimelineSettings{StartKeyID: start.ID, EndKeyID: end.ID, Zoom: av.TimelineZoomQuarter, ColorBy: av.TimelineColorByConditional}
	if stored.Views[0].Timeline.Settings != want {
		t.Fatalf("timeline settings were not saved: %+v", stored.Views[0].Timeline.Settings)
	}
	for _, invalid := range []any{nil, map[string]any{"zoom": "year"}, map[string]any{"startKeyID": "../start"}} {
		if err = setAttrViewTimeline(operation(invalid)); nil == err {
			t.Fatalf("invalid timeline settings accepted: %v", invalid)
		}
	}
	calendar := av.NewCalendarView()
	stored.Views = append(stored.Views, calendar)
	if err = av.SaveAttributeView(stored); nil != err {
		t.Fatal(err)
	}
	if err = setAttrViewTimeline(&Operation{AvID: attrView.ID, ViewID: calendar.ID, Data: next}); nil == err {
		t.Fatal("timeline settings must reject other layouts")
	}

	day := int64(86400000)
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	last := base + 3*day
	view.Timeline.Settings = av.TimelineSettings{StartKeyID: start.ID, Zoom: av.TimelineZoomWeek}
	values := map[string]*av.Value{}
	if err = setNewCalendarItemDate(attrView, "", view.ID, base, values, &last); nil != err {
		t.Fatal(err)
	}
	date := values[start.ID].Date
	if !date.IsNotEmpty || !date.IsNotTime || date.Content != base || !date.HasEndDate || !date.IsNotEmpty2 || date.Content2 != last {
		t.Fatalf("timeline creation without an end field must write the start date end: %+v", date)
	}
	values = map[string]*av.Value{}
	if err = setNewCalendarItemDate(attrView, "", view.ID, base, values, &base); nil != err || values[start.ID].Date.HasEndDate {
		t.Fatalf("a single day must not set an end date: %v", err)
	}

	view.Timeline.Settings.EndKeyID = end.ID
	values = map[string]*av.Value{"other": {Type: av.KeyTypeText, Text: &av.ValueText{Content: "template"}}}
	if err = setNewCalendarItemDate(attrView, "", view.ID, base, values, &last); nil != err {
		t.Fatal(err)
	}
	if values[start.ID].Date.HasEndDate || values[end.ID].Date.Content != last || values["other"].Text.Content != "template" {
		t.Fatal("timeline creation must write the end field and keep template values")
	}
	before := base - day
	if err = setNewCalendarItemDate(attrView, "", view.ID, base, map[string]*av.Value{}, &before); nil == err {
		t.Fatal("an end date before the start date must be rejected")
	}

	created := av.NewKey(ast.NewNodeID(), "Created", "", av.KeyTypeCreated)
	addAttributeViewKey(attrView, view, created, "")
	view.Timeline.Settings.StartKeyID = created.ID
	values = map[string]*av.Value{}
	if err = setNewCalendarItemDate(attrView, "", view.ID, base, values); nil == err || 0 != len(values) {
		t.Fatal("a system date start field must reject scheduling")
	}
	view.Timeline.Settings.StartKeyID = start.ID
	start.RenderTemplate = "2026-10-01"
	if err = setNewCalendarItemDate(attrView, "", view.ID, base, map[string]*av.Value{}); nil == err {
		t.Fatal("a computed start field must reject scheduling")
	}
	if err = setNewCalendarItemDate(stored, "", calendar.ID, base, map[string]*av.Value{}, &last); nil == err {
		t.Fatal("calendar items must reject an end date")
	}
}
