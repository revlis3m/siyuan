// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package apicontract

// revlis3m mods: Form view.

// AVFormField 是一个字段在表单中的设置；没有设置的可见字段按可选、无默认值处理。
type AVFormField struct {
	KeyID    string `json:"keyID"`
	Required bool   `json:"required"`
	// 默认值来源：空、value（固定值）、today（今天，日期字段）或 last（最近新增条目的值）。
	Default string `json:"default" api:"enum=|value|today|last"`
	// 固定默认值的文本形式：单选和状态为选项名，多选为逗号分隔的选项名，数字为数字文本，复选框为 true 或 false，关联为逗号分隔的条目 ID。
	Value string `json:"value"`
	// 显示条件：只有该单选、状态或复选框字段的值是 showWhenValues 之一时才显示本字段；空表示总是显示。
	ShowWhenKeyID  string   `json:"showWhenKeyID"`
	ShowWhenValues []string `json:"showWhenValues" api:"optional,nullable"`
}

// AVFormSettings 是 setAttrViewForm 事务的完整设置数据，也是表单布局的持久化设置。
// 表单的问题是视图中可见的字段，按视图字段顺序排列；提交时新增一个条目。
type AVFormSettings struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	SubmitLabel string         `json:"submitLabel"`
	Fields      []*AVFormField `json:"fields" api:"optional,nullable"`
}

type AVLayoutForm struct {
	*AVBaseLayout
	Columns []*AVViewTableColumn `json:"columns" api:"optional,nullable"`
	// 兼容旧版行顺序；空列表与底层布局一致省略，非空列表保持原有顺序。
	RowIDs   []string       `json:"rowIds,omitempty" api:"optional,nullable"`
	Settings AVFormSettings `json:"settings"`
}
