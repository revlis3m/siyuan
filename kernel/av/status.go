// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package av

import (
	"strings"

	"github.com/siyuan-note/siyuan/kernel/util"
)

// StatusGroup 描述状态字段选项所属的分组，按待办、进行中、已完成的顺序参与排序和分组。
type StatusGroup string

const (
	StatusGroupTodo       StatusGroup = "todo"
	StatusGroupInProgress StatusGroup = "inProgress"
	StatusGroupComplete   StatusGroup = "complete"
)

// StatusGroups 按显示和排序顺序返回全部状态分组。
func StatusGroups() []StatusGroup {
	return []StatusGroup{StatusGroupTodo, StatusGroupInProgress, StatusGroupComplete}
}

// IsValid 判断分组是否为已知取值，空值不是合法的显式分组。
func (g StatusGroup) IsValid() bool {
	return StatusGroupTodo == g || StatusGroupInProgress == g || StatusGroupComplete == g
}

// Normalize 将缺失或未知的分组视为待办，兼容没有分组的选项。
func (g StatusGroup) Normalize() StatusGroup {
	if g.IsValid() {
		return g
	}
	return StatusGroupTodo
}

// Rank 返回分组的排序序号：待办 < 进行中 < 已完成。
func (g StatusGroup) Rank() int {
	switch g.Normalize() {
	case StatusGroupInProgress:
		return 1
	case StatusGroupComplete:
		return 2
	default:
		return 0
	}
}

// IsSingleSelectKeyType 判断字段值是否最多只保存一个选项，状态字段与单选字段共用存储和编辑逻辑。
func IsSingleSelectKeyType(typ KeyType) bool {
	return KeyTypeSelect == typ || KeyTypeStatus == typ
}

// GetStatusGroup 返回选项所在分组，选项不存在或没有分组时视为待办。
func (k *Key) GetStatusGroup(name string) StatusGroup {
	if nil == k {
		return StatusGroupTodo
	}
	if option := k.GetOption(name); nil != option {
		return option.Group.Normalize()
	}
	return StatusGroupTodo
}

// GetOptionIndex 返回选项在字段中的位置，选项不存在时返回 -1。
func (k *Key) GetOptionIndex(name string) int {
	if nil == k {
		return -1
	}
	for i, option := range k.Options {
		if nil != option && option.Name == name {
			return i
		}
	}
	return -1
}

// StatusOptionRank 返回状态选项的排序序号，先按分组再按选项在字段中的顺序；空值排在最后。
func (k *Key) StatusOptionRank(name string) int {
	if "" == name {
		return 1 << 30
	}
	index := k.GetOptionIndex(name)
	if 0 > index {
		// 已删除的选项排在同组已有选项之后。
		index = 1 << 20
	}
	return k.GetStatusGroup(name).Rank()<<24 + index
}

// StatusOrderedOptions 按分组和字段内顺序返回状态选项。
func (k *Key) StatusOrderedOptions() (ret []*SelectOption) {
	if nil == k {
		return
	}
	for _, group := range StatusGroups() {
		for _, option := range k.Options {
			if nil != option && option.Group.Normalize() == group {
				ret = append(ret, option)
			}
		}
	}
	return
}

// IsStatusComplete 按状态字段的选项分组判断值是否已完成，空值视为未完成。
func (value *Value) IsStatusComplete(key *Key) bool {
	if nil == value || 1 > len(value.MSelect) || nil == value.MSelect[0] {
		return false
	}
	content := strings.TrimSpace(value.MSelect[0].Content)
	if "" == content {
		return false
	}
	return StatusGroupComplete == key.GetStatusGroup(content)
}

// InferStatusGroup 根据选项名称推断转换为状态字段时的分组，未识别的名称归入待办。
func InferStatusGroup(name string) StatusGroup {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "done", "complete", "completed", "terminé", "terminée", "fini", "finie":
		return StatusGroupComplete
	case "in progress", "doing", "en cours":
		return StatusGroupInProgress
	}
	return StatusGroupTodo
}

// NormalizeStatusOptions 为状态字段的选项补全分组；allowInfer 为真时按名称推断缺失分组，否则缺失分组视为待办。
func NormalizeStatusOptions(options []*SelectOption, allowInfer bool) {
	for _, option := range options {
		if nil == option {
			continue
		}
		if option.Group.IsValid() {
			continue
		}
		if allowInfer {
			option.Group = InferStatusGroup(option.Name)
		} else {
			option.Group = StatusGroupTodo
		}
	}
}

// NewDefaultStatusOptions 返回新建状态字段的默认选项：未开始（灰）、进行中（蓝）、已完成（绿）。
func NewDefaultStatusOptions() []*SelectOption {
	return []*SelectOption{
		{Name: statusI18n("statusNotStarted", "Not started"), Color: "5", Group: StatusGroupTodo},
		{Name: statusI18n("statusInProgress", "In progress"), Color: "6", Group: StatusGroupInProgress},
		{Name: statusI18n("statusDone", "Done"), Color: "11", Group: StatusGroupComplete},
	}
}

// statusI18n 读取属性视图语言包，语言包缺失时使用英文名称。
func statusI18n(key, fallback string) string {
	if langs, ok := util.AttrViewLangs[util.Lang]; ok {
		if text, ok := langs[key].(string); ok && "" != strings.TrimSpace(text) {
			return text
		}
	}
	return fallback
}

// IsValuelessStatusFilterOperator 判断是否为仅适用于状态字段且不需要比较值的筛选算子。
func IsValuelessStatusFilterOperator(operator FilterOperator) bool {
	return FilterOperatorIsComplete == operator || FilterOperatorIsNotComplete == operator
}
