// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// revlis3m mods: Form view settings.

import (
	"encoding/json"
	"fmt"

	"github.com/siyuan-note/siyuan/kernel/av"
)

func newAttributeViewFormLayout(attrView *av.AttributeView, fieldIDs []string) *av.LayoutForm {
	return &av.LayoutForm{
		LayoutTable: newAttributeViewListLayout(attrView, fieldIDs),
		Settings:    av.DefaultFormSettings(),
	}
}

func (tx *Transaction) doSetAttrViewForm(operation *Operation) *TxErr {
	if err := setAttrViewForm(operation); nil != err {
		return &TxErr{code: TxErrHandleAttributeView, id: operation.AvID, msg: err.Error()}
	}
	return nil
}

func setAttrViewForm(operation *Operation) error {
	if nil == operation.Data {
		return fmt.Errorf("form settings are required")
	}
	data, err := json.Marshal(operation.Data)
	if nil != err {
		return err
	}
	var settings av.FormSettings
	if err = json.Unmarshal(data, &settings); nil != err {
		return err
	}
	if nil == settings.Fields {
		settings.Fields = []*av.FormField{}
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
	if av.LayoutTypeForm != view.LayoutType || nil == view.Form {
		return av.ErrWrongLayoutType
	}
	view.Form.Settings = settings
	return av.SaveAttributeView(attrView)
}
