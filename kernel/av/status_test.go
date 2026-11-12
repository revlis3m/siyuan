// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package av

import (
	"encoding/json"
	"math"
	"slices"
	"testing"

	"github.com/siyuan-note/siyuan/kernel/util"
)

func newStatusTestKey() *Key {
	return &Key{ID: "status", Type: KeyTypeStatus, Options: []*SelectOption{
		{Name: "Done", Color: "11", Group: StatusGroupComplete},
		{Name: "Backlog", Color: "5"},
		{Name: "Doing", Color: "6", Group: StatusGroupInProgress},
		{Name: "Not started", Color: "5", Group: StatusGroupTodo},
		{Name: "Shipped", Color: "4", Group: StatusGroupComplete},
	}}
}

func newStatusTestValue(content string) *Value {
	value := &Value{KeyID: "status", Type: KeyTypeStatus, CreatedAt: 1, MSelect: []*ValueSelect{}}
	if "" != content {
		value.MSelect = []*ValueSelect{{Content: content}}
	}
	return value
}

func TestStatusGroupNormalizeAndRank(t *testing.T) {
	for _, test := range []struct {
		group StatusGroup
		want  StatusGroup
		rank  int
	}{
		{"", StatusGroupTodo, 0},
		{"unknown", StatusGroupTodo, 0},
		{StatusGroupTodo, StatusGroupTodo, 0},
		{StatusGroupInProgress, StatusGroupInProgress, 1},
		{StatusGroupComplete, StatusGroupComplete, 2},
	} {
		if got := test.group.Normalize(); got != test.want {
			t.Errorf("Normalize(%q) = %q, want %q", test.group, got, test.want)
		}
		if got := test.group.Rank(); got != test.rank {
			t.Errorf("Rank(%q) = %d, want %d", test.group, got, test.rank)
		}
	}
	key := newStatusTestKey()
	if got := key.GetStatusGroup("Backlog"); got != StatusGroupTodo {
		t.Fatalf("option without group = %q, want todo", got)
	}
	if got := key.GetStatusGroup("missing"); got != StatusGroupTodo {
		t.Fatalf("missing option = %q, want todo", got)
	}
}

func TestSelectOptionGroupJSONCompatibility(t *testing.T) {
	data, err := json.Marshal(&SelectOption{Name: "A", Color: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"name":"A","color":"1","desc":""}` {
		t.Fatalf("option without group changed its JSON shape: %s", data)
	}
	var option SelectOption
	if err = json.Unmarshal([]byte(`{"name":"A","color":"1","desc":"","group":"inProgress"}`), &option); err != nil {
		t.Fatal(err)
	}
	if option.Group != StatusGroupInProgress {
		t.Fatalf("group = %q, want inProgress", option.Group)
	}
}

func TestNewDefaultStatusOptions(t *testing.T) {
	oldLang, oldLangs := util.Lang, util.AttrViewLangs
	t.Cleanup(func() { util.Lang, util.AttrViewLangs = oldLang, oldLangs })

	util.Lang, util.AttrViewLangs = "en", map[string]map[string]any{}
	options := NewDefaultStatusOptions()
	want := []SelectOption{
		{Name: "Not started", Color: "5", Group: StatusGroupTodo},
		{Name: "In progress", Color: "6", Group: StatusGroupInProgress},
		{Name: "Done", Color: "11", Group: StatusGroupComplete},
	}
	if len(options) != len(want) {
		t.Fatalf("default options = %d, want %d", len(options), len(want))
	}
	for i, option := range options {
		if *option != want[i] {
			t.Fatalf("default option %d = %+v, want %+v", i, *option, want[i])
		}
	}

	util.Lang = "fr"
	util.AttrViewLangs = map[string]map[string]any{"fr": {
		"statusNotStarted": "Pas commencé", "statusInProgress": "En cours", "statusDone": "Terminé",
	}}
	options = NewDefaultStatusOptions()
	if options[0].Name != "Pas commencé" || options[1].Name != "En cours" || options[2].Name != "Terminé" {
		t.Fatalf("localized default options = %s, %s, %s", options[0].Name, options[1].Name, options[2].Name)
	}
}

func TestInferStatusGroup(t *testing.T) {
	for name, want := range map[string]StatusGroup{
		"Done": StatusGroupComplete, " complete ": StatusGroupComplete, "TERMINÉ": StatusGroupComplete,
		"fini": StatusGroupComplete, "In Progress": StatusGroupInProgress, "doing": StatusGroupInProgress,
		"En cours": StatusGroupInProgress, "Backlog": StatusGroupTodo, "": StatusGroupTodo,
	} {
		if got := InferStatusGroup(name); got != want {
			t.Errorf("InferStatusGroup(%q) = %q, want %q", name, got, want)
		}
	}

	options := []*SelectOption{{Name: "Done"}, {Name: "Doing"}, {Name: "Later"}, {Name: "Done twice", Group: StatusGroupInProgress}, nil}
	NormalizeStatusOptions(options, true)
	got := []StatusGroup{options[0].Group, options[1].Group, options[2].Group, options[3].Group}
	if !slices.Equal(got, []StatusGroup{StatusGroupComplete, StatusGroupInProgress, StatusGroupTodo, StatusGroupInProgress}) {
		t.Fatalf("inferred groups = %v", got)
	}
}

func TestStatusFilterIsComplete(t *testing.T) {
	key := newStatusTestKey()
	attrView := &AttributeView{KeyValues: []*KeyValues{{Key: key}}}
	for _, test := range []struct {
		content          string
		complete, active bool
	}{
		{"Done", true, false},
		{"Shipped", true, false},
		{"Doing", false, true},
		{"Backlog", false, true},
		{"", false, true},
		{"Removed option", false, true},
	} {
		value := newStatusTestValue(test.content)
		for operator, want := range map[FilterOperator]bool{
			FilterOperatorIsComplete:    test.complete,
			FilterOperatorIsNotComplete: test.active,
		} {
			filter := &ViewFilter{Column: key.ID, Operator: operator, Value: &Value{Type: KeyTypeStatus, MSelect: []*ValueSelect{}}}
			if got := value.Filter(filter, attrView, "item", nil, nil); got != want {
				t.Errorf("%s %q = %v, want %v", operator, test.content, got, want)
			}
			filter.Value = nil
			if got := value.Filter(filter, attrView, "item", nil, nil); got != want {
				t.Errorf("%s without value %q = %v, want %v", operator, test.content, got, want)
			}
		}
	}

	// 字段类型改变后筛选不生效。
	textValue := &Value{KeyID: key.ID, Type: KeyTypeText, Text: &ValueText{Content: "Done"}}
	filter := &ViewFilter{Column: key.ID, Operator: FilterOperatorIsComplete, Value: &Value{Type: KeyTypeStatus}}
	if !textValue.Filter(filter, attrView, "item", nil, nil) {
		t.Fatal("mismatched field type must not be filtered")
	}

	if evalMissingLeaf(&ViewFilter{Operator: FilterOperatorIsComplete}) {
		t.Fatal("missing status cell must not be complete")
	}
	if !evalMissingLeaf(&ViewFilter{Operator: FilterOperatorIsNotComplete}) {
		t.Fatal("missing status cell must be not complete")
	}
}

func TestStatusFilterSelectOperators(t *testing.T) {
	attrView := &AttributeView{KeyValues: []*KeyValues{{Key: newStatusTestKey()}}}
	filter := &ViewFilter{Column: "status", Value: &Value{Type: KeyTypeStatus, MSelect: []*ValueSelect{{Content: "Doing"}}}}
	for _, test := range []struct {
		operator FilterOperator
		content  string
		want     bool
	}{
		{FilterOperatorIsEqual, "Doing", true},
		{FilterOperatorIsEqual, "Done", false},
		{FilterOperatorIsNotEqual, "Done", true},
		{FilterOperatorIsNotEqual, "Doing", false},
		{FilterOperatorIsEmpty, "", true},
		{FilterOperatorIsEmpty, "Done", false},
		{FilterOperatorIsNotEmpty, "Done", true},
	} {
		filter.Operator = test.operator
		if got := newStatusTestValue(test.content).Filter(filter, attrView, "item", nil, nil); got != test.want {
			t.Errorf("%s %q = %v, want %v", test.operator, test.content, got, test.want)
		}
	}
	for _, operator := range []FilterOperator{FilterOperatorIsEqual, FilterOperatorIsNotEqual, FilterOperatorIsEmpty,
		FilterOperatorIsNotEmpty, FilterOperatorIsComplete, FilterOperatorIsNotComplete} {
		if !IsFilterOperatorAllowed(KeyTypeStatus, operator) {
			t.Errorf("status must accept %s", operator)
		}
	}
	for _, typ := range []KeyType{KeyTypeSelect, KeyTypeMSelect, KeyTypeText} {
		if IsFilterOperatorAllowed(typ, FilterOperatorIsComplete) || IsFilterOperatorAllowed(typ, FilterOperatorIsNotComplete) {
			t.Errorf("%s must not accept completion operators", typ)
		}
	}
}

func TestStatusFilterAffectValue(t *testing.T) {
	key := newStatusTestKey()
	for _, test := range []struct {
		operator FilterOperator
		want     string
	}{
		{FilterOperatorIsComplete, "Done"},
		{FilterOperatorIsNotComplete, "Backlog"},
		{FilterOperatorIsNotEmpty, "Backlog"},
	} {
		filter := &ViewFilter{Column: key.ID, Operator: test.operator, Value: &Value{Type: KeyTypeStatus, MSelect: []*ValueSelect{}}}
		value, _ := filter.GetAffectValue(key, "item")
		if nil == value || 1 != len(value.MSelect) || test.want != value.MSelect[0].Content {
			t.Fatalf("%s default value = %+v, want %s", test.operator, value, test.want)
		}
	}

	noComplete := &Key{ID: "status", Type: KeyTypeStatus, Options: []*SelectOption{{Name: "Todo"}}}
	filter := &ViewFilter{Column: "status", Operator: FilterOperatorIsComplete, Value: &Value{Type: KeyTypeStatus}}
	if value, _ := filter.GetAffectValue(noComplete, "item"); nil != value {
		t.Fatalf("missing complete option must not produce a default value: %+v", value)
	}
}

func TestStatusSortByGroupThenOptionOrder(t *testing.T) {
	key := newStatusTestKey()
	optionSort := OptionSortMap(key)
	ordered := []string{"Backlog", "Not started", "Doing", "Done", "Shipped"}
	for i := 1; i < len(ordered); i++ {
		if !(optionSort[ordered[i-1]] < optionSort[ordered[i]]) {
			t.Fatalf("%s must sort before %s: %v", ordered[i-1], ordered[i], optionSort)
		}
	}

	selectKey := &Key{ID: "select", Type: KeyTypeSelect, Options: key.Options}
	selectSort := OptionSortMap(selectKey)
	if selectSort["Done"] != 0 || selectSort["Shipped"] != 4 {
		t.Fatalf("select fields must keep plain option order: %v", selectSort)
	}

	contents := []string{"Shipped", "", "Doing", "Backlog", "Done", "Not started"}
	var rows []*TableRow
	for i, content := range contents {
		rows = append(rows, &TableRow{ID: content + string(rune('a'+i)), Cells: []*TableCell{{BaseValue: &BaseValue{
			ValueType: KeyTypeStatus, Value: newStatusTestValue(content)}}}})
	}
	for _, test := range []struct {
		order SortOrder
		want  []string
	}{
		{SortOrderAsc, []string{"Backlog", "Not started", "Doing", "Done", "Shipped", ""}},
		{SortOrderDesc, []string{"Shipped", "Done", "Doing", "Not started", "Backlog", ""}},
	} {
		table := &Table{
			BaseInstance: &BaseInstance{ID: "table", Sorts: []*ViewSort{{Column: key.ID, Order: test.order}}},
			Columns:      []*TableColumn{{BaseInstanceField: &BaseInstanceField{ID: key.ID, Type: KeyTypeStatus}}},
			Rows:         slices.Clone(rows),
		}
		Sort(table, &AttributeView{KeyValues: []*KeyValues{{Key: key}}})
		var got []string
		for _, row := range table.Rows {
			got = append(got, row.Cells[0].Value.String(false))
		}
		if !slices.Equal(got, test.want) {
			t.Fatalf("%s order = %v, want %v", test.order, got, test.want)
		}
	}
}

func TestStatusFieldCalc(t *testing.T) {
	key := newStatusTestKey()
	attrView := &AttributeView{KeyValues: []*KeyValues{{Key: key}}}
	column := &TableColumn{BaseInstanceField: &BaseInstanceField{ID: key.ID, Type: KeyTypeStatus, Calc: &FieldCalc{}}}
	table := &Table{BaseInstance: &BaseInstance{}, Columns: []*TableColumn{column}}
	for _, content := range []string{"Done", "Shipped", "Doing", "", "Backlog"} {
		table.Rows = append(table.Rows, &TableRow{Cells: []*TableCell{{BaseValue: &BaseValue{Value: newStatusTestValue(content)}}}})
	}
	table.Rows = append(table.Rows, &TableRow{Cells: []*TableCell{{BaseValue: &BaseValue{}}}})
	for _, test := range []struct {
		operator CalcOperator
		want     float64
		text     string
	}{
		{CalcOperatorCountComplete, 2, "2"},
		{CalcOperatorCountNotComplete, 4, "4"},
		{CalcOperatorPercentComplete, 2.0 / 6.0, "33.33%"},
		{CalcOperatorCountValues, 4, "4"},
		{CalcOperatorCountUniqueValues, 4, "4"},
		{CalcOperatorCountEmpty, 2, "2"},
	} {
		column.Calc = &FieldCalc{Operator: test.operator}
		Calc(table, attrView)
		result := column.Calc.Result
		if nil == result || nil == result.Number || math.Abs(result.Number.Content-test.want) > 1e-9 {
			t.Fatalf("%s = %+v, want %v", test.operator, result, test.want)
		}
		if got := result.Number.FormattedContent; got != test.text {
			t.Errorf("%s display = %q, want %q", test.operator, got, test.text)
		}
	}

	empty := &Table{BaseInstance: &BaseInstance{}, Columns: []*TableColumn{column}}
	column.Calc = &FieldCalc{Operator: CalcOperatorPercentComplete}
	Calc(empty, attrView)
	if nil != column.Calc.Result {
		t.Fatalf("percent complete of no items = %+v, want nil", column.Calc.Result)
	}
	for _, operator := range []CalcOperator{CalcOperatorCountComplete, CalcOperatorCountNotComplete, CalcOperatorPercentComplete} {
		if !isAVNumericCalcFilterOperator(operator) {
			t.Errorf("%s must be filtered as a number", operator)
		}
	}
}

func isAVNumericCalcFilterOperator(operator CalcOperator) bool {
	number, ok := calcFilterNumber[operator]
	return ok && number
}

func TestStatusRollupCalc(t *testing.T) {
	key := newStatusTestKey()
	for _, test := range []struct {
		operator CalcOperator
		want     float64
		text     string
	}{
		{CalcOperatorCountComplete, 2, "2"},
		{CalcOperatorCountNotComplete, 2, "2"},
		{CalcOperatorPercentComplete, 50, "50%"},
	} {
		rollup := &ValueRollup{}
		for _, content := range []string{"Done", "Doing", "Shipped", ""} {
			rollup.Contents = append(rollup.Contents, newStatusTestValue(content))
		}
		rollup.calcContents(&RollupCalc{Operator: test.operator}, key)
		if 1 != len(rollup.Contents) || nil == rollup.Contents[0].Number ||
			math.Abs(rollup.Contents[0].Number.Content-test.want) > 1e-9 {
			t.Fatalf("%s = %+v, want %v", test.operator, rollup.Contents, test.want)
		}
		if got := rollup.Contents[0].String(true); got != test.text {
			t.Errorf("%s display = %q, want %q", test.operator, got, test.text)
		}
	}

	rollup := &ValueRollup{}
	rollup.calcContents(&RollupCalc{Operator: CalcOperatorPercentComplete}, key)
	if 0 != len(rollup.Contents) {
		t.Fatal("empty relation must stay empty")
	}
}

func TestStatusRollupCountsUnsetTargetValues(t *testing.T) {
	key := newStatusTestKey()
	dest := &AttributeView{ID: "dest", KeyValues: []*KeyValues{{Key: key, Values: []*Value{
		{ID: "20240101000000-aaaaaaa", KeyID: key.ID, BlockID: "a", Type: KeyTypeStatus, MSelect: []*ValueSelect{{Content: "Done"}}},
	}}}}
	relation := &Value{Type: KeyTypeRelation, Relation: &ValueRelation{BlockIDs: []string{"a", "b"}}}
	rollup := &ValueRollup{}
	rollup.BuildContents(dest, key, relation, &RollupCalc{Operator: CalcOperatorPercentComplete}, nil)
	if 1 != len(rollup.Contents) || nil == rollup.Contents[0].Number || 50 != rollup.Contents[0].Number.Content {
		t.Fatalf("percent complete with an unset target = %+v, want 50", rollup.Contents)
	}

	rollup = &ValueRollup{}
	rollup.BuildContents(dest, key, relation, &RollupCalc{Operator: CalcOperatorCountValues}, nil)
	if 1 != len(rollup.Contents) || 1 != rollup.Contents[0].Number.Content {
		t.Fatalf("count values must ignore unset targets: %+v", rollup.Contents)
	}
}

func TestStatusValueStorageMatchesSelect(t *testing.T) {
	value := newStatusTestValue("Doing")
	if "Doing" != value.String(false) || value.IsEmpty() {
		t.Fatalf("status value = %q, empty %v", value.String(false), value.IsEmpty())
	}
	defaultValue := GetAttributeViewDefaultValue("20240101000000-aaaaaaa", "status", "item", KeyTypeStatus, false)
	if nil == defaultValue.MSelect || 0 != len(defaultValue.MSelect) || !defaultValue.IsEmpty() {
		t.Fatalf("default status value = %+v", defaultValue)
	}
	if !IsSelectKeyType(KeyTypeStatus) || !IsSingleSelectKeyType(KeyTypeStatus) || IsSingleSelectKeyType(KeyTypeMSelect) {
		t.Fatal("status must share single-select storage")
	}
}

// revlis3m mods: automation conditions accept the status completion operators and match by option group.
func TestAutomationStatusCompletionFilters(t *testing.T) {
	key := newStatusTestKey()
	values := []*Value{}
	for itemID, content := range map[string]string{"done": "Done", "shipped": "Shipped", "doing": "Doing", "empty": ""} {
		value := newStatusTestValue(content)
		value.BlockID = itemID
		values = append(values, value)
	}
	text := &Key{ID: "text", Type: KeyTypeText}
	view := &AttributeView{ID: "tasks", KeyValues: []*KeyValues{{Key: key, Values: values}, {Key: text}}}

	for _, operator := range []FilterOperator{FilterOperatorIsComplete, FilterOperatorIsNotComplete} {
		for name, value := range map[string]*Value{"with value": {Type: KeyTypeStatus, MSelect: []*ValueSelect{}}, "without value": nil} {
			filters := []*ViewFilter{{Column: key.ID, Operator: operator, Value: value}}
			if err := ValidateAutomationFilters(view, filters); err != nil {
				t.Fatalf("%s %s rejected: %v", operator, name, err)
			}
			for itemID, complete := range map[string]bool{"done": true, "shipped": true, "doing": false, "empty": false} {
				got, err := view.MatchesAutomationFilters(itemID, filters, nil)
				if err != nil {
					t.Fatalf("%s %s on %s: %v", operator, name, itemID, err)
				}
				if want := complete == (FilterOperatorIsComplete == operator); got != want {
					t.Errorf("%s %s on %s = %v, want %v", operator, name, itemID, got, want)
				}
			}
		}
	}

	// Only status fields have groups: the operators stay refused on other field types.
	for _, operator := range []FilterOperator{FilterOperatorIsComplete, FilterOperatorIsNotComplete} {
		filters := []*ViewFilter{{Column: text.ID, Operator: operator, Value: &Value{Type: KeyTypeText, Text: &ValueText{}}}}
		if err := ValidateAutomationFilters(view, filters); err == nil {
			t.Errorf("%s accepted on a text field", operator)
		}
	}
	// Other operators on a status field still need a value.
	if err := ValidateAutomationFilters(view, []*ViewFilter{{Column: key.ID, Operator: FilterOperatorContains}}); err == nil {
		t.Error("Contains without a value accepted")
	}
}
