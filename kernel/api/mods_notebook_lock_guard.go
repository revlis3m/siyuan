// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

// revlis3m mods: side doors of the notebook PIN lock (see model/mods_notebook_lock.go).
//
// A PIN-locked notebook is closed, so the editor no longer shows it. Its files, its history and its databases
// are still on disk, and several routes can read them by path or ID without opening the notebook: the file API,
// document history, WebDAV, database rendering… This guard refuses any request that names a locked notebook or
// one of its databases, in its address or in its body, wherever the route, and the few routes that read the whole
// workspace at once. Routes that only manage the lock or that store interface state are let through.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/siyuan/kernel/model"
)

// notebookPINGuardMaxBody is the largest request body the guard reads to look for a locked notebook.
const notebookPINGuardMaxBody = 8 << 20

// Bodies of these routes are not inspected: they manage the lock itself, or they store interface state that
// can name a notebook (layout, local storage, settings) without reading any of its content.
var notebookPINGuardOpenPaths = map[string]bool{
	"/api/notebook/getNotebookPINStatus":  true,
	"/api/notebook/setNotebookPIN":        true,
	"/api/notebook/removeNotebookPIN":     true,
	"/api/notebook/unlockNotebookPIN":     true,
	"/api/notebook/lockNotebookPIN":       true,
	"/api/notebook/setNotebookPINMinutes": true,
	"/api/notebook/touchNotebookPIN":      true,
	"/api/notebook/openNotebook":          true, // model.Mount refuses it with its own message
	"/api/notebook/closeNotebook":         true,
	"/api/notebook/lsNotebooks":           true,
}

// These routes read or restore the whole workspace (full export, data snapshots), so they would carry a locked
// notebook's content without naming it. They are refused while any PIN notebook is locked.
var notebookPINGuardWholeDataPaths = map[string]bool{
	"/api/export/exportData":             true,
	"/api/export/exportDataInFolder":     true,
	"/api/repo/checkoutRepo":             true,
	"/api/repo/diffRepoSnapshots":        true,
	"/api/repo/searchRepoFile":           true,
	"/api/repo/getRepoDocHistory":        true,
	"/api/repo/openRepoSnapshotFile":     true,
	"/api/repo/rollbackRepoSnapshotFile": true,
	"/api/repo/exportRepoFile":           true,
	"/api/repo/getRepoFile":              true,
}

const notebookPINGuardWholeDataMessage = "not available while a notebook is locked with a PIN: unlock it first"

var notebookPINGuardOpenPrefixes = []string{
	"/api/storage/", "/api/system/", "/api/setting/", "/api/notification/", "/api/sync/", "/api/cloud/", "/api/account/",
}

// NotebookPINGuard is registered before every route, static ones included.
func NotebookPINGuard(c *gin.Context) {
	terms := model.LockedNotebookPINTerms()
	if len(terms) == 0 {
		c.Next()
		return
	}

	path := c.Request.URL.Path
	if notebookPINGuardOpenPaths[path] {
		c.Next()
		return
	}
	if notebookPINGuardWholeDataPaths[path] {
		notebookPINGuardRefuseWith(c, notebookPINGuardWholeDataMessage)
		return
	}
	query, err := url.QueryUnescape(c.Request.URL.RawQuery)
	if err != nil {
		query = c.Request.URL.RawQuery
	}
	if notebookPINGuardMentions(path, terms) || notebookPINGuardMentions(query, terms) {
		notebookPINGuardRefuse(c)
		return
	}

	inspectBody := c.Request.Body != nil && c.Request.Body != http.NoBody &&
		(strings.HasPrefix(path, "/api/") || path == "/mcp" || strings.HasPrefix(path, "/mcp/"))
	for _, prefix := range notebookPINGuardOpenPrefixes {
		if strings.HasPrefix(path, prefix) {
			inspectBody = false
		}
	}
	if !inspectBody {
		c.Next()
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, notebookPINGuardMaxBody+1))
	if err != nil {
		notebookPINGuardRefuse(c)
		return
	}
	// Give the handlers the body back: what was read, then whatever is left of a larger one.
	c.Request.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(body), c.Request.Body), c.Request.Body}

	isJSON := strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "json") || json.Valid(body)
	if len(body) > notebookPINGuardMaxBody {
		// Too large to check fully. JSON can hide an ID behind escapes, so it is refused; other bodies
		// (file uploads) are checked on what was read.
		if isJSON || notebookPINGuardMentions(string(body), terms) {
			notebookPINGuardRefuse(c)
			return
		}
		c.Next()
		return
	}
	if notebookPINGuardBodyMentions(body, terms) {
		notebookPINGuardRefuse(c)
		return
	}
	c.Next()
}

func notebookPINGuardMentions(text string, terms map[string]string) bool {
	if text == "" {
		return false
	}
	for term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

// notebookPINGuardBodyMentions looks for a locked ID in the body as sent and, for JSON, in every decoded key
// and string value, so that an ID written with JSON escapes is found too.
func notebookPINGuardBodyMentions(body []byte, terms map[string]string) bool {
	if notebookPINGuardMentions(string(body), terms) {
		return true
	}
	var decoded any
	if json.Unmarshal(body, &decoded) != nil {
		return false
	}
	return notebookPINGuardValueMentions(decoded, terms)
}

func notebookPINGuardValueMentions(value any, terms map[string]string) bool {
	switch typed := value.(type) {
	case string:
		return notebookPINGuardMentions(typed, terms)
	case []any:
		for _, item := range typed {
			if notebookPINGuardValueMentions(item, terms) {
				return true
			}
		}
	case map[string]any:
		for key, item := range typed {
			if notebookPINGuardMentions(key, terms) || notebookPINGuardValueMentions(item, terms) {
				return true
			}
		}
	}
	return false
}

func notebookPINGuardRefuse(c *gin.Context) {
	notebookPINGuardRefuseWith(c, model.ErrNotebookPINLocked.Error())
}

func notebookPINGuardRefuseWith(c *gin.Context, message string) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.AbortWithStatusJSON(http.StatusOK, gin.H{"code": -1, "msg": message, "data": nil})
		return
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": -1, "msg": message, "data": nil})
}
