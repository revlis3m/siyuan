// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"testing"

	"github.com/siyuan-note/siyuan/kernel/av"
)

func assertStatusOptionGroups(t *testing.T, key *av.Key, expected map[string]av.StatusGroup) {
	t.Helper()
	if len(key.Options) != len(expected) {
		t.Fatalf("expected %d options, got %d", len(expected), len(key.Options))
	}
	for name, group := range expected {
		option := key.GetOption(name)
		if nil == option {
			t.Fatalf("option [%s] not found", name)
		}
		if option.Group != group {
			t.Fatalf("option [%s] group = %q, want %q", name, option.Group, group)
		}
	}
}

func TestNewAttributeViewKeyStatusDefaults(t *testing.T) {
	setupAttributeViewValidationTest(t)

	key, err := newAttributeViewKey("20261101000000-status", "Status", string(av.KeyTypeStatus), "", av.DateDisplayFormatFull)
	if nil != err {
		t.Fatalf("create status key failed: %s", err)
	}
	assertSelectOptionNames(t, key.Options, []string{"Not started", "In progress", "Done"})
	assertStatusOptionGroups(t, key, map[string]av.StatusGroup{
		"Not started": av.StatusGroupTodo, "In progress": av.StatusGroupInProgress, "Done": av.StatusGroupComplete,
	})
	if "5" != key.Options[0].Color || "6" != key.Options[1].Color || "11" != key.Options[2].Color {
		t.Fatalf("default status colors = %s/%s/%s", key.Options[0].Color, key.Options[1].Color, key.Options[2].Color)
	}

	selectKey, err := newAttributeViewKey("20261101000000-select", "Select", string(av.KeyTypeSelect), "", av.DateDisplayFormatFull)
	if nil != err || 0 != len(selectKey.Options) {
		t.Fatalf("select key must start without options: %v %v", selectKey, err)
	}
}

func TestUpdateAttributeViewColumnConvertsBetweenSelectAndStatus(t *testing.T) {
	setupAttributeViewValidationTest(t)

	attrView := av.NewAttributeView("20261101000001-status")
	selectKey := attrView.KeyValues[1].Key
	selectKey.Options = []*av.SelectOption{
		{Name: "Backlog", Color: "1"},
		{Name: "In Progress", Color: "2"},
		{Name: "Terminé", Color: "3"},
		{Name: "Done", Color: "4"},
	}
	attrView.KeyValues[1].Values = []*av.Value{{
		ID: "20261101000001-vvvvvvv", KeyID: selectKey.ID, BlockID: "20261101000001-iiiiiii", Type: av.KeyTypeSelect,
		MSelect: []*av.ValueSelect{{Content: "Done", Color: "4"}},
	}}
	if err := av.SaveAttributeView(attrView); nil != err {
		t.Fatalf("save attribute view failed: %s", err)
	}

	convert := func(typ av.KeyType) *av.KeyValues {
		t.Helper()
		if err := updateAttributeViewColumn(&Operation{AvID: attrView.ID, ID: selectKey.ID, Name: selectKey.Name, Typ: string(typ)}); nil != err {
			t.Fatalf("convert to %s failed: %s", typ, err)
		}
		parsed, err := av.ParseAttributeView(attrView.ID)
		if nil != err {
			t.Fatalf("parse attribute view failed: %s", err)
		}
		keyValues, err := parsed.GetKeyValues(selectKey.ID)
		if nil != err {
			t.Fatalf("get key values failed: %s", err)
		}
		if typ != keyValues.Key.Type || 1 != len(keyValues.Values) || typ != keyValues.Values[0].Type ||
			"Done" != keyValues.Values[0].String(false) {
			t.Fatalf("conversion to %s lost the key type or value: %+v", typ, keyValues.Values)
		}
		return keyValues
	}

	expected := map[string]av.StatusGroup{
		"Backlog": av.StatusGroupTodo, "In Progress": av.StatusGroupInProgress,
		"Terminé": av.StatusGroupComplete, "Done": av.StatusGroupComplete,
	}
	keyValues := convert(av.KeyTypeStatus)
	assertSelectOptionNames(t, keyValues.Key.Options, []string{"Backlog", "In Progress", "Terminé", "Done"})
	assertStatusOptionGroups(t, keyValues.Key, expected)
	if !keyValues.Values[0].IsStatusComplete(keyValues.Key) {
		t.Fatal("converted value must be complete")
	}

	// 转回单选保留选项，分组被忽略但保留，再次转为状态时沿用已有分组。
	keyValues = convert(av.KeyTypeSelect)
	assertSelectOptionNames(t, keyValues.Key.Options, []string{"Backlog", "In Progress", "Terminé", "Done"})
	parsed, _ := av.ParseAttributeView(attrView.ID)
	parsedKey, _ := parsed.GetKey(selectKey.ID)
	parsedKey.Options[0].Group = av.StatusGroupInProgress
	if err := av.SaveAttributeView(parsed); nil != err {
		t.Fatalf("save attribute view failed: %s", err)
	}
	keyValues = convert(av.KeyTypeStatus)
	expected["Backlog"] = av.StatusGroupInProgress
	assertStatusOptionGroups(t, keyValues.Key, expected)
}

func TestUpdateAttributeViewColumnConvertsTextToStatusWithDefaults(t *testing.T) {
	setupAttributeViewValidationTest(t)

	attrView := av.NewAttributeView("20261101000002-status")
	key := attrView.KeyValues[1].Key
	key.Type = av.KeyTypeText
	if err := av.SaveAttributeView(attrView); nil != err {
		t.Fatalf("save attribute view failed: %s", err)
	}
	if err := updateAttributeViewColumn(&Operation{AvID: attrView.ID, ID: key.ID, Name: key.Name, Typ: string(av.KeyTypeStatus)}); nil != err {
		t.Fatalf("convert to status failed: %s", err)
	}
	parsed, err := av.ParseAttributeView(attrView.ID)
	if nil != err {
		t.Fatalf("parse attribute view failed: %s", err)
	}
	statusKey, _ := parsed.GetKey(key.ID)
	assertStatusOptionGroups(t, statusKey, map[string]av.StatusGroup{
		"Not started": av.StatusGroupTodo, "In progress": av.StatusGroupInProgress, "Done": av.StatusGroupComplete,
	})
}

func TestUpdateAttributeViewColumnOptionsStatusGroup(t *testing.T) {
	setupAttributeViewValidationTest(t)

	attrView := av.NewAttributeView("20261101000003-status")
	statusKey := attrView.KeyValues[1].Key
	statusKey.Type = av.KeyTypeStatus
	statusKey.Options = av.NewDefaultStatusOptions()
	if err := av.SaveAttributeView(attrView); nil != err {
		t.Fatalf("save attribute view failed: %s", err)
	}

	update := func(options []*av.SelectOption) *av.Key {
		t.Helper()
		if err := updateAttributeViewColumnOptions(&Operation{AvID: attrView.ID, ID: statusKey.ID, Data: options}); nil != err {
			t.Fatalf("update options failed: %s", err)
		}
		parsed, err := av.ParseAttributeView(attrView.ID)
		if nil != err {
			t.Fatalf("parse attribute view failed: %s", err)
		}
		key, _ := parsed.GetKey(statusKey.ID)
		return key
	}

	// 显式分组移动选项，缺省分组保持原分组，新选项缺省进入待办。
	key := update([]*av.SelectOption{
		{Name: "Not started", Color: "5"},
		{Name: "In progress", Color: "7", Group: av.StatusGroupComplete},
		{Name: "Done", Color: "11", Group: "invalid"},
		{Name: "Blocked", Color: "1"},
		{Name: "Review", Color: "2", Group: av.StatusGroupInProgress},
	})
	assertStatusOptionGroups(t, key, map[string]av.StatusGroup{
		"Not started": av.StatusGroupTodo, "In progress": av.StatusGroupComplete, "Done": av.StatusGroupComplete,
		"Blocked": av.StatusGroupTodo, "Review": av.StatusGroupInProgress,
	})
	if "7" != key.GetOption("In progress").Color {
		t.Fatal("option color was not updated together with its group")
	}

	if err := UpdateAttributeViewKeyConfig(attrView.ID, statusKey.ID, map[string]any{
		"optionUpdate": map[string]any{"name": "Blocked", "group": "inProgress"},
	}); nil != err {
		t.Fatalf("optionUpdate group failed: %s", err)
	}
	parsed, _ := av.ParseAttributeView(attrView.ID)
	key, _ = parsed.GetKey(statusKey.ID)
	if av.StatusGroupInProgress != key.GetOption("Blocked").Group || "1" != key.GetOption("Blocked").Color {
		t.Fatalf("optionUpdate did not move the option: %+v", key.GetOption("Blocked"))
	}
	if err := UpdateAttributeViewKeyConfig(attrView.ID, statusKey.ID, map[string]any{
		"optionUpdate": map[string]any{"name": "Blocked", "group": "later"},
	}); nil == err {
		t.Fatal("invalid status group must be rejected")
	}
	if err := UpdateAttributeViewKeyConfig(attrView.ID, statusKey.ID, map[string]any{
		"options": []any{map[string]any{"name": "Waiting", "group": "unknown"}},
	}); nil == err {
		t.Fatal("invalid status group in options must be rejected")
	}
}

func TestUpdateAttributeViewKeyConfigRejectsGroupOnSelect(t *testing.T) {
	setupAttributeViewValidationTest(t)

	attrView := av.NewAttributeView("20261101000004-status")
	selectKey := attrView.KeyValues[1].Key
	selectKey.Options = []*av.SelectOption{{Name: "A", Color: "1"}}
	if err := av.SaveAttributeView(attrView); nil != err {
		t.Fatalf("save attribute view failed: %s", err)
	}
	if err := UpdateAttributeViewKeyConfig(attrView.ID, selectKey.ID, map[string]any{
		"optionUpdate": map[string]any{"name": "A", "group": "complete"},
	}); nil == err {
		t.Fatal("select options must not accept a status group")
	}

	// 单选字段的批量更新忽略分组字段。
	if err := updateAttributeViewColumnOptions(&Operation{AvID: attrView.ID, ID: selectKey.ID,
		Data: []*av.SelectOption{{Name: "A", Color: "2", Group: av.StatusGroupComplete}, {Name: "B", Group: av.StatusGroupComplete}}}); nil != err {
		t.Fatalf("update options failed: %s", err)
	}
	parsed, _ := av.ParseAttributeView(attrView.ID)
	key, _ := parsed.GetKey(selectKey.ID)
	if "" != key.GetOption("A").Group || "" != key.GetOption("B").Group {
		t.Fatal("select options must not store a status group")
	}
}
