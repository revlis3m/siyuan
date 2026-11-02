// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"github.com/siyuan-note/siyuan/kernel/apicontract"
	"github.com/siyuan-note/siyuan/kernel/av"
)

func toContractAVTimelineSettings(value *av.TimelineSettings) *apicontract.AVTimelineSettings {
	if nil == value {
		return nil
	}
	return &apicontract.AVTimelineSettings{StartKeyID: value.StartKeyID, EndKeyID: value.EndKeyID,
		Zoom: string(value.Zoom), ColorBy: value.ColorBy}
}

func toContractAVLayoutTimeline(value *av.LayoutTimeline) *apicontract.AVLayoutTimeline {
	if nil == value {
		return nil
	}
	return &apicontract.AVLayoutTimeline{
		AVBaseLayout: toContractAVBaseLayout(value.BaseLayout),
		Columns:      avContractSlice(value.Columns, toContractAVViewTableColumn),
		RowIDs:       value.RowIDs,
		Settings:     *toContractAVTimelineSettings(&value.Settings),
	}
}

func toContractAVTimelineItem(value *av.TimelineItem) *apicontract.AVTimelineItem {
	if nil == value {
		return nil
	}
	return &apicontract.AVTimelineItem{ID: value.ID, Start: value.Start, End: value.End, IsNotTime: value.IsNotTime,
		Invalid: value.Invalid, Title: value.Title, BlockID: value.BlockID, Color: toContractAVValueSelect(value.Color)}
}
