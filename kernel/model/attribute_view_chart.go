// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"encoding/json"
	"fmt"

	"github.com/siyuan-note/siyuan/kernel/av"
)

// newAttributeViewChartLayout 创建图表布局：字段沿用列表的默认显隐（导出时只输出主键），图表设置使用默认分组字段。
func newAttributeViewChartLayout(attrView *av.AttributeView, fieldIDs []string) *av.LayoutChart {
	return &av.LayoutChart{
		LayoutTable: newAttributeViewListLayout(attrView, fieldIDs),
		Settings:    av.NewChartSettings(attrView),
	}
}

func (tx *Transaction) doSetAttrViewChart(operation *Operation) *TxErr {
	if err := setAttrViewChart(operation); nil != err {
		return &TxErr{code: TxErrHandleAttributeView, id: operation.AvID, msg: err.Error()}
	}
	return nil
}

func setAttrViewChart(operation *Operation) error {
	if nil == operation.Data {
		return fmt.Errorf("chart settings are required")
	}
	data, err := json.Marshal(operation.Data)
	if nil != err {
		return err
	}
	var settings av.ChartSettings
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
	if av.LayoutTypeChart != view.LayoutType || nil == view.Chart {
		return av.ErrWrongLayoutType
	}
	// 字段绑定独立保存，缺失或类型变化由渲染器按未绑定处理，撤销可以恢复此前的绑定。
	view.Chart.Settings = settings
	return av.SaveAttributeView(attrView)
}
