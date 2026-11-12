// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import "testing"

func TestNotebookPINGuardBodyMentions(t *testing.T) {
	const box, database = "20261107194007-ferwma2", "20261107194100-avavava"
	terms := map[string]string{box: box, database: box}
	for name, test := range map[string]struct {
		body string
		want bool
	}{
		"file path":            {`{"path":"/data/20261107194007-ferwma2/20261107194200-abcdefg.sy"}`, true},
		"history path":         {`{"historyPath":"/siyuan/workspace/history/2026-11-07-200000-update/20261107194007-ferwma2/a.sy"}`, true},
		"database id":          {`{"id":"20261107194100-avavava"}`, true},
		"database file":        {`{"path":"/data/storage/av/20261107194100-avavava.json"}`, true},
		"escaped id":           {`{"path":"/data/\u0032\u0030261107194007-ferwma2/a.sy"}`, true},
		"escaped key":          {`{"\u0032\u0030261107194007-ferwma2":true}`, true},
		"nested":               {`{"doOperations":[{"data":{"ids":["x","20261107194100-avavava"]}}]}`, true},
		"multipart":            {"--b\r\nContent-Disposition: form-data; name=\"path\"\r\n\r\n/data/20261107194007-ferwma2/a.sy\r\n--b--", true},
		"another notebook":     {`{"path":"/data/20261107194007-zzzzzzz/a.sy"}`, false},
		"another database":     {`{"id":"20261107194100-bbbbbbb"}`, false},
		"empty":                {``, false},
		"not json, no mention": {`plain text`, false},
	} {
		if got := notebookPINGuardBodyMentions([]byte(test.body), terms); got != test.want {
			t.Errorf("%s: got %v, want %v", name, got, test.want)
		}
	}
	if notebookPINGuardBodyMentions([]byte(`{"id":"20261107194100-avavava"}`), nil) {
		t.Error("nothing is refused when nothing is locked")
	}
	if !notebookPINGuardMentions("/history/2026-11-07-200000-update/"+box+"/a.sy", terms) {
		t.Error("an address naming the notebook is refused")
	}
}

func TestNotebookPINGuardWholeDataPaths(t *testing.T) {
	for _, path := range []string{"/api/export/exportData", "/api/repo/openRepoSnapshotFile", "/api/repo/getRepoFile"} {
		if !notebookPINGuardWholeDataPaths[path] {
			t.Errorf("%s must be refused while a notebook is locked", path)
		}
	}
	for _, path := range []string{"/api/repo/createSnapshot", "/api/notebook/unlockNotebookPIN", "/api/export/exportMd"} {
		if notebookPINGuardWholeDataPaths[path] {
			t.Errorf("%s must stay available", path)
		}
	}
}
