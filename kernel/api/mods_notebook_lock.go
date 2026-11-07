// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

// revlis3m mods: notebook PIN lock (see model/mods_notebook_lock.go).

import (
	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/siyuan/kernel/apicontract"
	"github.com/siyuan-note/siyuan/kernel/model"
)

var getNotebookPINStatus = contractHandler(apicontract.GetNotebookPINStatus, func(c *gin.Context, request apicontract.EmptyRequest) apicontract.Response[apicontract.NotebookPINStatusData] {
	boxes, minutes := model.GetNotebookPINStatus()
	data := apicontract.NotebookPINStatusData{Notebooks: []*apicontract.NotebookPINState{}, Minutes: minutes}
	for _, box := range boxes {
		data.Notebooks = append(data.Notebooks, &apicontract.NotebookPINState{Notebook: box.Notebook, Locked: box.Locked})
	}
	return apicontract.Success(data)
})

var setNotebookPIN = contractHandler(apicontract.SetNotebookPIN, func(c *gin.Context, request apicontract.SetNotebookPINRequest) apicontract.Response[apicontract.Null] {
	if err := model.SetNotebookPIN(request.Notebook, request.CurrentPIN, request.PIN); err != nil {
		return apicontract.Failure[apicontract.Null](-1, err.Error())
	}
	return apicontract.Success(apicontract.Null{})
})

var removeNotebookPIN = contractHandler(apicontract.RemoveNotebookPIN, func(c *gin.Context, request apicontract.NotebookPINRequest) apicontract.Response[apicontract.Null] {
	if err := model.RemoveNotebookPIN(request.Notebook, request.PIN); err != nil {
		return apicontract.Failure[apicontract.Null](-1, err.Error())
	}
	return apicontract.Success(apicontract.Null{})
})

var unlockNotebookPIN = contractHandler(apicontract.UnlockNotebookPIN, func(c *gin.Context, request apicontract.NotebookPINRequest) apicontract.Response[apicontract.Null] {
	if err := model.UnlockNotebookPIN(request.Notebook, request.PIN); err != nil {
		return apicontract.Failure[apicontract.Null](-1, err.Error())
	}
	return apicontract.Success(apicontract.Null{})
})

var lockNotebookPIN = contractHandler(apicontract.LockNotebookPIN, func(c *gin.Context, request apicontract.NotebookIDRequest) apicontract.Response[apicontract.Null] {
	if err := model.LockNotebookPIN(request.Notebook); err != nil {
		return apicontract.Failure[apicontract.Null](-1, err.Error())
	}
	return apicontract.Success(apicontract.Null{})
})

var setNotebookPINMinutes = contractHandler(apicontract.SetNotebookPINMinutes, func(c *gin.Context, request apicontract.NotebookPINMinutesRequest) apicontract.Response[apicontract.Null] {
	if err := model.SetNotebookPINMinutes(request.Minutes); err != nil {
		return apicontract.Failure[apicontract.Null](-1, err.Error())
	}
	return apicontract.Success(apicontract.Null{})
})
