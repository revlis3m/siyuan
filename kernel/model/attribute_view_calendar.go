// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/88250/lute/ast"
	"github.com/siyuan-note/siyuan/kernel/av"
)

func newAttributeViewCalendarLayout(attrView *av.AttributeView, fieldIDs []string) *av.LayoutCalendar {
	ret := &av.LayoutCalendar{
		LayoutTable: newAttributeViewListLayout(attrView, fieldIDs),
		Settings:    av.CalendarSettings{WeekStart: 1},
	}
	for _, keyType := range av.DateKeyTypes() {
		for _, keyValues := range attrView.KeyValues {
			if keyValues.Key.Type == keyType {
				ret.Settings.DateKeyID = keyValues.Key.ID
				return ret
			}
		}
	}
	return ret
}

func (tx *Transaction) doSetAttrViewCalendar(operation *Operation) *TxErr {
	if err := setAttrViewCalendar(operation); nil != err {
		return &TxErr{code: TxErrHandleAttributeView, id: operation.AvID, msg: err.Error()}
	}
	return nil
}

func setAttrViewCalendar(operation *Operation) error {
	if nil == operation.Data {
		return fmt.Errorf("calendar settings are required")
	}
	data, err := json.Marshal(operation.Data)
	if nil != err {
		return err
	}
	var settings av.CalendarSettings
	if err = json.Unmarshal(data, &settings); nil != err {
		return err
	}
	if settings.WeekStart < 0 || settings.WeekStart > 6 {
		return fmt.Errorf("invalid calendar week start")
	}
	if !settings.ValidRowLimit() {
		return fmt.Errorf("invalid calendar row limit")
	}
	attrView, err := av.ParseAttributeView(operation.AvID)
	if nil != err {
		return err
	}
	view, err := getAttrViewOperationView(attrView, operation)
	if nil != err {
		return err
	}
	if av.LayoutTypeCalendar != view.LayoutType || nil == view.Calendar {
		return av.ErrWrongLayoutType
	}
	// 字段绑定独立保存，缺失或类型变化由渲染器显示设置提示，撤销可以恢复此前的绑定。
	for _, id := range []string{settings.DateKeyID, settings.ColorKeyID} {
		if id == "" {
			continue
		}
		if !ast.IsNodeIDPattern(id) {
			return fmt.Errorf("invalid calendar field [%s]", id)
		}
	}
	view.Calendar.Settings = settings
	return av.SaveAttributeView(attrView)
}

// setNewCalendarItemDate 将选中的日期合入模板字段，与创建条目共用一个事务。
// 时间线视图按开始日期和可选的结束日期写入，日历视图不接受结束日期。
func setNewCalendarItemDate(attrView *av.AttributeView, blockID, viewID string, date int64, values map[string]*av.Value, endDates ...*int64) error {
	view, err := getAttrViewOperationView(attrView, &Operation{BlockID: blockID, ViewID: viewID})
	if nil != err {
		return err
	}
	var endDate *int64
	if len(endDates) > 0 {
		endDate = endDates[0]
	}
	if view.LayoutType == av.LayoutTypeTimeline {
		return setNewTimelineItemDates(attrView, view, date, endDate, values)
	}
	if nil != endDate {
		return fmt.Errorf("calendar items do not accept an end date")
	}
	if view.LayoutType != av.LayoutTypeCalendar || nil == view.Calendar || date < -62135596800000-14*3600000 || date > 253402300799999+14*3600000 {
		return fmt.Errorf("invalid calendar item date")
	}
	key, err := attrView.GetKey(view.Calendar.Settings.DateKeyID)
	if nil != err || key.Type != av.KeyTypeDate || strings.TrimSpace(key.RenderTemplate) != "" {
		return fmt.Errorf("calendar date field is not editable")
	}
	values[key.ID] = &av.Value{Type: av.KeyTypeDate, KeyID: key.ID,
		Date: &av.ValueDate{Content: date, IsNotEmpty: true, IsNotTime: true}}
	return nil
}
