// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/siyuan-note/siyuan/kernel/av"
)

// newAttributeViewTimelineLayout 创建时间线布局：左侧只显示主键，其余字段默认隐藏，字段绑定使用默认设置。
func newAttributeViewTimelineLayout(attrView *av.AttributeView, fieldIDs []string) *av.LayoutTimeline {
	return &av.LayoutTimeline{
		LayoutTable: newAttributeViewListLayout(attrView, fieldIDs),
		Settings:    av.NewTimelineSettings(attrView),
	}
}

func (tx *Transaction) doSetAttrViewTimeline(operation *Operation) *TxErr {
	if err := setAttrViewTimeline(operation); nil != err {
		return &TxErr{code: TxErrHandleAttributeView, id: operation.AvID, msg: err.Error()}
	}
	return nil
}

func setAttrViewTimeline(operation *Operation) error {
	if nil == operation.Data {
		return fmt.Errorf("timeline settings are required")
	}
	data, err := json.Marshal(operation.Data)
	if nil != err {
		return err
	}
	var settings av.TimelineSettings
	if err = json.Unmarshal(data, &settings); nil != err {
		return err
	}
	if err = settings.Validate(); nil != err {
		return err
	}
	attrView, err := av.ParseAttributeView(operation.AvID)
	if nil != err {
		return err
	}
	view, err := getAttrViewOperationView(attrView, operation)
	if nil != err {
		return err
	}
	if av.LayoutTypeTimeline != view.LayoutType || nil == view.Timeline {
		return av.ErrWrongLayoutType
	}
	// 字段绑定独立保存，缺失或类型变化由渲染器按未绑定处理，撤销可以恢复此前的绑定。
	view.Timeline.Settings = settings
	return av.SaveAttributeView(attrView)
}

const (
	attributeViewMinItemDate = -62135596800000 - 14*3600000
	attributeViewMaxItemDate = 253402300799999 + 14*3600000
)

// setNewTimelineItemDates 将新条目的开始和结束日期合入模板字段，与创建条目共用一个事务。
// 结束日期写入结束字段；未绑定结束字段时写入开始日期的结束日期，与开始日期相同时不设置结束日期。
func setNewTimelineItemDates(attrView *av.AttributeView, view *av.View, start int64, end *int64, values map[string]*av.Value) error {
	if nil == view.Timeline || start < attributeViewMinItemDate || start > attributeViewMaxItemDate ||
		(nil != end && (*end < start || *end > attributeViewMaxItemDate)) {
		return fmt.Errorf("invalid timeline item date")
	}
	key, err := attrView.GetKey(view.Timeline.Settings.StartKeyID)
	if nil != err || key.Type != av.KeyTypeDate || strings.TrimSpace(key.RenderTemplate) != "" {
		return fmt.Errorf("timeline start field is not editable")
	}
	startDate := &av.ValueDate{Content: start, IsNotEmpty: true, IsNotTime: true}
	if nil == end || *end == start {
		values[key.ID] = &av.Value{Type: av.KeyTypeDate, KeyID: key.ID, Date: startDate}
		return nil
	}
	endKeyID := view.Timeline.Settings.EndKeyID
	if "" != endKeyID && endKeyID != key.ID {
		endKey, endErr := attrView.GetKey(endKeyID)
		if nil == endErr && endKey.Type == av.KeyTypeDate && strings.TrimSpace(endKey.RenderTemplate) == "" {
			values[key.ID] = &av.Value{Type: av.KeyTypeDate, KeyID: key.ID, Date: startDate}
			values[endKey.ID] = &av.Value{Type: av.KeyTypeDate, KeyID: endKey.ID,
				Date: &av.ValueDate{Content: *end, IsNotEmpty: true, IsNotTime: true}}
			return nil
		}
		return fmt.Errorf("timeline end field is not editable")
	}
	startDate.HasEndDate, startDate.Content2, startDate.IsNotEmpty2 = true, *end, true
	values[key.ID] = &av.Value{Type: av.KeyTypeDate, KeyID: key.ID, Date: startDate}
	return nil
}
