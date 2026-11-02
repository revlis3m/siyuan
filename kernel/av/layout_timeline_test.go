// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package av

import (
	"testing"
)

const (
	timelineTestStartKey  = "20261101000000-tstart1"
	timelineTestEndKey    = "20261101000000-tend001"
	timelineTestStatusKey = "20261101000000-tstatus"
	timelineTestSelectKey = "20261101000000-tselect"
	timelineTestBlockKey  = "20261101000000-tblock1"
	timelineTestDay       = int64(86400000)
	timelineTestStart     = int64(1790006400000)
)

func TestTimelineSettingsValidate(t *testing.T) {
	valid := []TimelineSettings{
		{},
		{StartKeyID: timelineTestStartKey, EndKeyID: timelineTestEndKey, Zoom: TimelineZoomQuarter, ColorBy: timelineTestStatusKey},
		{StartKeyID: timelineTestStartKey, Zoom: TimelineZoomDay, ColorBy: TimelineColorByConditional},
	}
	for _, settings := range valid {
		if err := settings.Validate(); nil != err {
			t.Fatalf("valid timeline settings %+v rejected: %v", settings, err)
		}
	}
	invalid := []TimelineSettings{
		{Zoom: "year"},
		{StartKeyID: "start"},
		{EndKeyID: "../end"},
		{ColorBy: "color"},
	}
	for _, settings := range invalid {
		if err := settings.Validate(); nil == err {
			t.Fatalf("invalid timeline settings %+v accepted", settings)
		}
	}
}

func TestTimelineDefaultSettings(t *testing.T) {
	attrView := &AttributeView{KeyValues: []*KeyValues{
		{Key: &Key{ID: timelineTestBlockKey, Type: KeyTypeBlock}},
		{Key: &Key{ID: timelineTestSelectKey, Type: KeyTypeSelect}},
		{Key: &Key{ID: "20261101000000-tcreate", Type: KeyTypeCreated}},
		{Key: &Key{ID: timelineTestStartKey, Type: KeyTypeDate}},
		{Key: &Key{ID: timelineTestStatusKey, Type: KeyTypeStatus}},
	}}
	settings := NewTimelineSettings(attrView)
	if settings.StartKeyID != timelineTestStartKey || settings.EndKeyID != "" || settings.Zoom != TimelineZoomWeek ||
		settings.ColorBy != timelineTestStatusKey {
		t.Fatalf("unexpected default timeline settings: %+v", settings)
	}

	attrView.KeyValues = attrView.KeyValues[:3]
	settings = NewTimelineSettings(attrView)
	if settings.StartKeyID != "20261101000000-tcreate" || settings.ColorBy != timelineTestSelectKey {
		t.Fatalf("timeline must fall back to a system date and select colors: %+v", settings)
	}

	attrView.KeyValues = attrView.KeyValues[:1]
	settings = NewTimelineSettings(attrView)
	if settings.StartKeyID != "" || settings.ColorBy != "" || settings.Zoom != TimelineZoomWeek {
		t.Fatalf("timeline without date or option fields must stay unbound: %+v", settings)
	}
}

func TestTimelineIntervalResolution(t *testing.T) {
	day := func(n int64) int64 { return timelineTestStart + n*timelineTestDay }
	cases := []struct {
		name               string
		start, end         *ValueDate
		wantStart, wantEnd int64
		invalid, ok        bool
	}{
		{"empty", &ValueDate{}, nil, 0, 0, false, false},
		{"missing", nil, nil, 0, 0, false, false},
		{"single", &ValueDate{Content: day(0), IsNotEmpty: true, IsNotTime: true}, nil, day(0), day(0), false, true},
		{"content2", &ValueDate{Content: day(0), IsNotEmpty: true, HasEndDate: true, Content2: day(3), IsNotEmpty2: true}, nil, day(0), day(3), false, true},
		{"content2 disabled", &ValueDate{Content: day(0), IsNotEmpty: true, Content2: day(3), IsNotEmpty2: true}, nil, day(0), day(0), false, true},
		{"end field wins", &ValueDate{Content: day(0), IsNotEmpty: true, HasEndDate: true, Content2: day(3), IsNotEmpty2: true},
			&ValueDate{Content: day(5), IsNotEmpty: true}, day(0), day(5), false, true},
		{"empty end field falls back", &ValueDate{Content: day(0), IsNotEmpty: true, HasEndDate: true, Content2: day(2), IsNotEmpty2: true},
			&ValueDate{}, day(0), day(2), false, true},
		{"reversed", &ValueDate{Content: day(4), IsNotEmpty: true}, &ValueDate{Content: day(1), IsNotEmpty: true}, day(4), day(4), true, true},
		{"only end date", &ValueDate{HasEndDate: true, Content2: day(2), IsNotEmpty2: true}, nil, day(2), day(2), false, true},
	}
	for _, test := range cases {
		start, end, _, invalid, ok := ResolveTimelineInterval(test.start, test.end)
		if ok != test.ok || invalid != test.invalid || (ok && (start != test.wantStart || end != test.wantEnd)) {
			t.Fatalf("%s: got start=%d end=%d invalid=%t ok=%t", test.name, start, end, invalid, ok)
		}
	}
}

func newTimelineTestRow(id string, values ...*Value) *TableRow {
	row := &TableRow{ID: id, Cells: []*TableCell{{BaseValue: &BaseValue{ID: id + "b", ValueType: KeyTypeBlock, Value: &Value{
		KeyID: timelineTestBlockKey, BlockID: id, Type: KeyTypeBlock, IsDetached: true, Block: &ValueBlock{Content: "Item " + id},
	}}}}}
	for _, value := range values {
		row.Cells = append(row.Cells, &TableCell{BaseValue: &BaseValue{ID: value.KeyID + id, ValueType: value.Type, Value: value}})
	}
	return row
}

func TestTimelineRowsScheduledUnscheduledAndColors(t *testing.T) {
	column := func(id string, typ KeyType, options ...*SelectOption) *TableColumn {
		return &TableColumn{BaseInstanceField: &BaseInstanceField{ID: id, Type: typ, Options: options}}
	}
	date := func(days int64) *Value {
		return &Value{KeyID: timelineTestStartKey, Type: KeyTypeDate, Date: &ValueDate{
			Content: timelineTestStart + days*timelineTestDay, IsNotEmpty: true, IsNotTime: true}}
	}
	status := func(name, color string) *Value {
		return &Value{KeyID: timelineTestStatusKey, Type: KeyTypeStatus, MSelect: []*ValueSelect{{Content: name, Color: color}}}
	}
	ranged := date(1)
	ranged.Date.HasEndDate, ranged.Date.Content2, ranged.Date.IsNotEmpty2 = true, timelineTestStart+4*timelineTestDay, true
	conditional := newTimelineTestRow("20261101000004-ccccccc", date(6))
	conditional.ConditionalColors = &ItemConditionalColors{Background: &ValueSelect{Color: "3"}}
	timeline := &Timeline{Table: &Table{
		BaseInstance: &BaseInstance{ID: "20261101000000-tview01"},
		Columns: []*TableColumn{
			column(timelineTestBlockKey, KeyTypeBlock),
			column(timelineTestStartKey, KeyTypeDate),
			column(timelineTestStatusKey, KeyTypeStatus, &SelectOption{Name: "Doing", Color: "6"}),
		},
		Rows: []*TableRow{
			newTimelineTestRow("20261101000001-aaaaaaa", ranged, status("Doing", "1")),
			newTimelineTestRow("20261101000002-bbbbbbb", &Value{KeyID: timelineTestStartKey, Type: KeyTypeDate, Date: &ValueDate{}}),
			newTimelineTestRow("20261101000003-ddddddd", date(0), status("Removed", "9")),
			conditional,
			newTimelineTestRow("20261101000005-eeeeeee"),
		},
		Timeline: &TimelineSettings{StartKeyID: timelineTestStartKey, ColorBy: timelineTestStatusKey},
	}}
	ResolveTimelineRows(timeline)
	if len(timeline.Rows) != 3 || len(timeline.TimelineItems) != 3 || timeline.RowCount != 3 ||
		len(timeline.TimelineUnscheduled) != 2 {
		t.Fatalf("unexpected split: rows=%d items=%d unscheduled=%d", len(timeline.Rows), len(timeline.TimelineItems), len(timeline.TimelineUnscheduled))
	}
	first := timeline.TimelineItems[0]
	if first.ID != "20261101000001-aaaaaaa" || first.Start != timelineTestStart+timelineTestDay ||
		first.End != timelineTestStart+4*timelineTestDay || !first.IsNotTime || first.Title != "Item 20261101000001-aaaaaaa" ||
		first.BlockID != "" || nil == first.Color || first.Color.Color != "6" || first.Color.Content != "Doing" {
		t.Fatalf("unexpected first timeline item: %+v %+v", first, first.Color)
	}
	if second := timeline.TimelineItems[1]; second.Color == nil || second.Color.Color != "9" || second.End != second.Start {
		t.Fatalf("missing options must keep the stored selection color: %+v", second)
	}
	if timeline.TimelineItems[2].Color != nil {
		t.Fatal("option colors must ignore conditional colors")
	}
	if timeline.TimelineUnscheduled[0].ID != "20261101000002-bbbbbbb" || timeline.TimelineUnscheduled[1].ID != "20261101000005-eeeeeee" {
		t.Fatal("unscheduled rows must keep the view order")
	}
	for i, row := range timeline.Rows {
		if row.ID != timeline.TimelineItems[i].ID {
			t.Fatal("timeline items must follow the row order")
		}
	}

	timeline.Rows = append(timeline.Rows, timeline.TimelineUnscheduled...)
	timeline.Timeline.ColorBy = TimelineColorByConditional
	ResolveTimelineRows(timeline)
	for _, item := range timeline.TimelineItems {
		if (item.ID == conditional.ID) != (nil != item.Color && item.Color.Color == "3") {
			t.Fatalf("conditional color mismatch: %+v", item)
		}
	}

	RetainTimelineRows(timeline.Table, func(row *TableRow) bool {
		return row.ID != "20261101000001-aaaaaaa" && row.ID != "20261101000005-eeeeeee"
	})
	if len(timeline.Rows) != 2 || len(timeline.TimelineItems) != 2 || timeline.RowCount != 2 || len(timeline.TimelineUnscheduled) != 1 {
		t.Fatal("publish filtering must remove rows, intervals and unscheduled rows together")
	}

	timeline.Rows = append(timeline.Rows, timeline.TimelineUnscheduled...)
	timeline.Timeline.StartKeyID = timelineTestStatusKey
	ResolveTimelineRows(timeline)
	if len(timeline.Rows) != 0 || len(timeline.TimelineItems) != 0 || len(timeline.TimelineUnscheduled) != 3 {
		t.Fatal("a non-date start field must list every row as unscheduled")
	}
}

func TestTimelineEndFieldAndLayoutRegistration(t *testing.T) {
	end := &Value{KeyID: timelineTestEndKey, Type: KeyTypeDate, Date: &ValueDate{Content: timelineTestStart + 2*timelineTestDay, IsNotEmpty: true}}
	start := &Value{KeyID: timelineTestStartKey, Type: KeyTypeCreated, Created: &ValueCreated{Content: timelineTestStart, IsNotEmpty: true}}
	timeline := &Timeline{Table: &Table{
		BaseInstance: &BaseInstance{ID: "20261101000000-tview02"},
		Columns: []*TableColumn{
			{BaseInstanceField: &BaseInstanceField{ID: timelineTestStartKey, Type: KeyTypeCreated}},
			{BaseInstanceField: &BaseInstanceField{ID: timelineTestEndKey, Type: KeyTypeDate}},
		},
		Rows:     []*TableRow{newTimelineTestRow("20261101000006-fffffff", start, end)},
		Timeline: &TimelineSettings{StartKeyID: timelineTestStartKey, EndKeyID: timelineTestEndKey},
	}}
	ResolveTimelineRows(timeline)
	if len(timeline.TimelineItems) != 1 || timeline.TimelineItems[0].End != timelineTestStart+2*timelineTestDay ||
		timeline.TimelineItems[0].IsNotTime {
		t.Fatalf("end field must close the interval: %+v", timeline.TimelineItems)
	}
	if timeline.GetType() != LayoutTypeTimeline || TableFromViewable(timeline) != timeline.Table {
		t.Fatal("timeline must expose its table rows")
	}

	view := &View{ID: "20261101000000-tview03", LayoutType: LayoutTypeTimeline}
	database := &AttributeView{Views: []*View{view}}
	if err := database.ValidateLayouts(); nil == err {
		t.Fatal("missing timeline layout must fail validation")
	}
	view.Timeline = &LayoutTimeline{LayoutTable: NewLayoutTable(), Settings: TimelineSettings{Zoom: "year"}}
	if err := database.ValidateLayouts(); nil == err {
		t.Fatal("invalid timeline zoom must fail validation")
	}
	view.Timeline.Settings.Zoom = TimelineZoomMonth
	if err := database.ValidateLayouts(); nil != err {
		t.Fatalf("valid timeline rejected: %v", err)
	}
	if view.GetTableLayout() != view.Timeline.LayoutTable || view.GetBaseLayout() != view.Timeline.BaseLayout ||
		view.TableLayouts()[len(view.TableLayouts())-1] != view.Timeline.LayoutTable {
		t.Fatal("timeline must share the row and column layout helpers")
	}
	view.Group = &ViewGroup{Field: timelineTestStatusKey}
	if view.IsGroupView() || !KeepsPrimaryKeyVisible(LayoutTypeTimeline) {
		t.Fatal("timeline must not group items and must keep its primary key visible")
	}
}
