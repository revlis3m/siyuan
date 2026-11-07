// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package av

import "testing"

func TestFormSettingsValidate(t *testing.T) {
	key := "20261107120000-abcdefg"
	other := "20261107120001-abcdefg"
	valid := FormSettings{Title: "Add", Fields: []*FormField{
		{KeyID: key, Required: true, Default: FormDefaultToday},
		{KeyID: other, Default: FormDefaultValue, Value: "Expense", ShowWhenKeyID: key, ShowWhenValues: []string{"Transfer"}},
	}}
	if err := valid.Validate(); nil != err {
		t.Fatalf("valid settings: %v", err)
	}
	for name, settings := range map[string]FormSettings{
		"bad key":        {Fields: []*FormField{{KeyID: "x"}}},
		"duplicate":      {Fields: []*FormField{{KeyID: key}, {KeyID: key}}},
		"bad default":    {Fields: []*FormField{{KeyID: key, Default: "tomorrow"}}},
		"self condition": {Fields: []*FormField{{KeyID: key, ShowWhenKeyID: key}}},
		"nil field":      {Fields: []*FormField{nil}},
	} {
		if err := settings.Validate(); nil == err {
			t.Fatalf("%s should be rejected", name)
		}
	}
}

func TestResolveFormKeepsLatestValues(t *testing.T) {
	key, ignored := "20261107120000-abcdefg", "20261107120001-abcdefg"
	row := func(id, value string) *TableRow {
		return &TableRow{ID: id, Cells: []*TableCell{
			{BaseValue: &BaseValue{Value: &Value{KeyID: key, Type: KeyTypeText, Text: &ValueText{Content: value}}}},
			{BaseValue: &BaseValue{Value: &Value{KeyID: ignored, Type: KeyTypeText, Text: &ValueText{Content: "x"}}}},
		}}
	}
	form := &Form{Table: &Table{
		Rows:         []*TableRow{row("20261107100000-aaaaaaa", "old"), row("20261107110000-aaaaaaa", "new"), row("20261107090000-aaaaaaa", "older")},
		FormSettings: &FormSettings{Fields: []*FormField{{KeyID: key, Default: FormDefaultLast}}},
	}}
	ResolveForm(form)
	if 0 != len(form.Rows) || 3 != form.RowCount {
		t.Fatalf("rows are dropped and counted: %d %d", len(form.Rows), form.RowCount)
	}
	if value := form.FormLast[key]; nil == value || "new" != value.Text.Content {
		t.Fatalf("latest value: %+v", value)
	}
	if _, ok := form.FormLast[ignored]; ok {
		t.Fatal("only fields with the last default are returned")
	}
}

func TestFormViewLayoutRegistration(t *testing.T) {
	view := &View{ID: "20261107120002-abcdefg", LayoutType: LayoutTypeForm,
		Form: &LayoutForm{LayoutTable: NewLayoutTable(), Settings: DefaultFormSettings()}}
	if view.GetTableLayout() != view.Form.LayoutTable || nil == view.GetBaseLayout() {
		t.Fatal("form uses the table layout")
	}
	if !KeepsPrimaryKeyVisible(LayoutTypeForm) || view.IsGroupView() {
		t.Fatal("form keeps the primary key and has no groups")
	}
	attrView := &AttributeView{Views: []*View{view}}
	if err := attrView.ValidateFormLayouts(); nil != err {
		t.Fatal(err)
	}
	view.Form = nil
	if err := attrView.ValidateFormLayouts(); nil == err {
		t.Fatal("missing form layout is rejected")
	}
}

func TestFormInputKeyTypes(t *testing.T) {
	for _, keyType := range []KeyType{KeyTypeBlock, KeyTypeNumber, KeyTypeDate, KeyTypeSelect, KeyTypeRelation, KeyTypeCheckbox} {
		if !IsFormInputKeyType(keyType) {
			t.Fatalf("%s can be answered", keyType)
		}
	}
	for _, keyType := range []KeyType{KeyTypeTemplate, KeyTypeRollup, KeyTypeCreated, KeyTypeUpdated, KeyTypeMAsset} {
		if IsFormInputKeyType(keyType) {
			t.Fatalf("%s is not asked in a form", keyType)
		}
	}
}
