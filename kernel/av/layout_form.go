// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package av

// revlis3m mods: Form view. The view's visible fields, in order, are the questions; submitting adds an item.

import (
	"fmt"
	"unicode/utf8"

	"github.com/88250/lute/ast"
)

// FormDefault 是表单字段的默认值来源。
type FormDefault string

const (
	FormDefaultNone  FormDefault = ""      // 空
	FormDefaultValue FormDefault = "value" // 固定值，见 FormField.Value
	FormDefaultToday FormDefault = "today" // 今天（日期字段）
	FormDefaultLast  FormDefault = "last"  // 最近新增条目的值
)

func (d FormDefault) IsValid() bool {
	switch d {
	case FormDefaultNone, FormDefaultValue, FormDefaultToday, FormDefaultLast:
		return true
	}
	return false
}

const (
	formMaxText   = 4096
	formMaxFields = 512
)

// FormField 保存一个字段在表单中的设置；没有设置的字段按可选、无默认值处理。
type FormField struct {
	KeyID    string      `json:"keyID"`    // 字段
	Required bool        `json:"required"` // 必填
	Default  FormDefault `json:"default"`  // 默认值来源
	// Value 固定默认值的文本形式：单选和状态为选项名，多选为逗号分隔的选项名，数字为数字文本，复选框为 true 或 false，
	// 关联为逗号分隔的条目 ID，其余为文本。
	Value string `json:"value"`
	// ShowWhenKeyID 和 ShowWhenValues 设置显示条件：只有该单选、状态或复选框字段的值是其中之一时才显示本字段。
	ShowWhenKeyID  string   `json:"showWhenKeyID"`
	ShowWhenValues []string `json:"showWhenValues"`
}

// FormSettings 保存表单视图的显示设置，也是 setAttrViewForm 事务的完整数据。
type FormSettings struct {
	Title       string       `json:"title"`       // 表单标题，空时使用视图名称
	Description string       `json:"description"` // 标题下方的说明
	SubmitLabel string       `json:"submitLabel"` // 提交按钮文字，空时使用默认文字
	Fields      []*FormField `json:"fields"`      // 字段设置
}

func DefaultFormSettings() FormSettings {
	return FormSettings{Fields: []*FormField{}}
}

func (s FormSettings) Validate() error {
	if utf8.RuneCountInString(s.Title) > formMaxText || utf8.RuneCountInString(s.Description) > formMaxText ||
		utf8.RuneCountInString(s.SubmitLabel) > formMaxText {
		return fmt.Errorf("form text too long")
	}
	if len(s.Fields) > formMaxFields {
		return fmt.Errorf("too many form fields")
	}
	seen := map[string]bool{}
	for _, field := range s.Fields {
		if nil == field || !ast.IsNodeIDPattern(field.KeyID) || seen[field.KeyID] {
			return fmt.Errorf("invalid form field")
		}
		seen[field.KeyID] = true
		if !field.Default.IsValid() {
			return fmt.Errorf("invalid form default [%s]", field.Default)
		}
		if "" != field.ShowWhenKeyID && (!ast.IsNodeIDPattern(field.ShowWhenKeyID) || field.ShowWhenKeyID == field.KeyID) {
			return fmt.Errorf("invalid form condition field [%s]", field.ShowWhenKeyID)
		}
		if utf8.RuneCountInString(field.Value) > formMaxText || len(field.ShowWhenValues) > formMaxFields {
			return fmt.Errorf("form field value too long")
		}
	}
	return nil
}

func (s *FormSettings) remapKeyIDs(keyIDMap map[string]string) {
	for _, field := range s.Fields {
		if nil == field {
			continue
		}
		if id := keyIDMap[field.KeyID]; id != "" {
			field.KeyID = id
		}
		if id := keyIDMap[field.ShowWhenKeyID]; id != "" {
			field.ShowWhenKeyID = id
		}
	}
}

// LayoutForm 是表单布局，字段顺序和显隐复用行列布局：可见字段按顺序成为表单问题。
type LayoutForm struct {
	*LayoutTable
	Settings FormSettings `json:"settings"`
}

// Form 描述表单视图实例：只返回字段、设置和最近条目的值，不返回条目。
type Form struct {
	*Table
}

func (*Form) GetType() LayoutType { return LayoutTypeForm }

// IsFormInputKeyType reports whether a field can be answered in a form (computed and file fields cannot).
func IsFormInputKeyType(keyType KeyType) bool {
	switch keyType {
	case KeyTypeBlock, KeyTypeText, KeyTypeURL, KeyTypeEmail, KeyTypePhone, KeyTypeNumber, KeyTypeDate, KeyTypeSelect,
		KeyTypeStatus, KeyTypeMSelect, KeyTypeCheckbox, KeyTypeRelation:
		return true
	}
	return false
}

func NewFormView() *View {
	view := NewTableView()
	view.Name = GetAttributeViewI18n("form")
	view.LayoutType = LayoutTypeForm
	view.Form = &LayoutForm{LayoutTable: view.Table, Settings: DefaultFormSettings()}
	view.Table = nil
	return view
}

// ResolveForm 在筛选之后取最近新增条目的字段值，供"上次的值"默认值使用，然后去掉条目。
func ResolveForm(form *Form) {
	if nil == form || nil == form.Table {
		return
	}
	// 条目 ID 以创建时间开头，最大的 ID 就是最近新增的条目。
	var last *TableRow
	for _, row := range form.Rows {
		if nil != row && (nil == last || row.ID > last.ID) {
			last = row
		}
	}
	form.FormLast = map[string]*Value{}
	if nil != last && nil != form.FormSettings {
		wanted := map[string]bool{}
		for _, field := range form.FormSettings.Fields {
			if nil != field && FormDefaultLast == field.Default {
				wanted[field.KeyID] = true
			}
		}
		for _, cell := range last.Cells {
			if nil != cell && nil != cell.Value && wanted[cell.Value.KeyID] {
				form.FormLast[cell.Value.KeyID] = cell.Value
			}
		}
	}
	form.RowCount = len(form.Rows)
	form.Rows = []*TableRow{}
}

func (attrView *AttributeView) ValidateFormLayouts() error {
	var validate func(*View) error
	validate = func(view *View) error {
		if nil == view {
			return nil
		}
		if LayoutTypeForm == view.LayoutType && nil == view.Form {
			return fmt.Errorf("form layout missing in view [%s]", view.ID)
		}
		if layout := view.Form; nil != layout {
			if nil == layout.LayoutTable || nil == layout.BaseLayout || layout.Spec != 0 || nil != layout.Settings.Validate() {
				return fmt.Errorf("invalid form layout in view [%s]", view.ID)
			}
			for _, column := range layout.Columns {
				if nil == column || nil == column.BaseField {
					return fmt.Errorf("invalid form field in view [%s]", view.ID)
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
