// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

// revlis3m mods: Form view contract mapping.

import (
	"github.com/siyuan-note/siyuan/kernel/apicontract"
	"github.com/siyuan-note/siyuan/kernel/av"
)

func toContractAVFormSettings(value *av.FormSettings) *apicontract.AVFormSettings {
	if nil == value {
		return nil
	}
	ret := &apicontract.AVFormSettings{Title: value.Title, Description: value.Description, SubmitLabel: value.SubmitLabel,
		Fields: []*apicontract.AVFormField{}}
	for _, field := range value.Fields {
		if nil == field {
			continue
		}
		ret.Fields = append(ret.Fields, &apicontract.AVFormField{KeyID: field.KeyID, Required: field.Required,
			Default: string(field.Default), Value: field.Value, ShowWhenKeyID: field.ShowWhenKeyID,
			ShowWhenValues: field.ShowWhenValues})
	}
	return ret
}

func toContractAVLayoutForm(value *av.LayoutForm) *apicontract.AVLayoutForm {
	if nil == value {
		return nil
	}
	return &apicontract.AVLayoutForm{
		AVBaseLayout: toContractAVBaseLayout(value.BaseLayout),
		Columns:      avContractSlice(value.Columns, toContractAVViewTableColumn),
		RowIDs:       value.RowIDs,
		Settings:     *toContractAVFormSettings(&value.Settings),
	}
}
