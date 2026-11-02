// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"github.com/siyuan-note/siyuan/kernel/apicontract"
	"github.com/siyuan-note/siyuan/kernel/av"
)

func toContractAVChartSettings(value *av.ChartSettings) *apicontract.AVChartSettings {
	if nil == value {
		return nil
	}
	return &apicontract.AVChartSettings{
		ChartType:       string(value.ChartType),
		XKeyID:          value.XKeyID,
		DateBucket:      string(value.DateBucket),
		Measure:         string(value.Measure),
		ValueKeyID:      value.ValueKeyID,
		StackKeyID:      value.StackKeyID,
		Sort:            string(value.Sort),
		ShowEmptyGroups: value.ShowEmptyGroups,
		ShowLegend:      value.ShowLegend,
		ShowDataLabels:  value.ShowDataLabels,
		Height:          value.Height,
	}
}

func toContractAVLayoutChart(value *av.LayoutChart) *apicontract.AVLayoutChart {
	if nil == value {
		return nil
	}
	return &apicontract.AVLayoutChart{
		AVBaseLayout: toContractAVBaseLayout(value.BaseLayout),
		Columns:      avContractSlice(value.Columns, toContractAVViewTableColumn),
		RowIDs:       value.RowIDs,
		Settings:     *toContractAVChartSettings(&value.Settings),
	}
}

func toContractAVChartData(value *av.ChartData) *apicontract.AVChartData {
	if nil == value {
		return nil
	}
	return &apicontract.AVChartData{
		Measure:    string(value.Measure),
		XType:      string(value.XType),
		StackType:  string(value.StackType),
		Categories: avContractSlice(value.Categories, toContractAVChartCategory),
		Series:     avContractSlice(value.Series, toContractAVChartSeries),
		Total:      value.Total,
		ItemCount:  value.ItemCount,
	}
}

func toContractAVChartCategory(value *av.ChartCategory) *apicontract.AVChartCategory {
	if nil == value {
		return nil
	}
	return &apicontract.AVChartCategory{Key: value.Key, Label: value.Label, Color: value.Color,
		ResolvedColor: toContractAVAttributeViewColor(value.ResolvedColor), Empty: value.Empty, Start: value.Start,
		Value: value.Value, Count: value.Count}
}

func toContractAVChartSeries(value *av.ChartSeries) *apicontract.AVChartSeries {
	if nil == value {
		return nil
	}
	return &apicontract.AVChartSeries{Key: value.Key, Label: value.Label, Color: value.Color,
		ResolvedColor: toContractAVAttributeViewColor(value.ResolvedColor), Empty: value.Empty, Values: value.Values,
		Counts: value.Counts, Items: avContractSlice(value.Items, func(items []*av.ChartItem) []*apicontract.AVChartItem {
			return avContractSlice(items, toContractAVChartItem)
		})}
}

func toContractAVChartItem(value *av.ChartItem) *apicontract.AVChartItem {
	if nil == value {
		return nil
	}
	return &apicontract.AVChartItem{ID: value.ID, Title: value.Title, BlockID: value.BlockID, ValueID: value.ValueID,
		IsDetached: value.IsDetached}
}
