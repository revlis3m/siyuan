// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"errors"
	"testing"
	"time"

	"github.com/siyuan-note/siyuan/kernel/util"
)

func resetNotebookPINForTest(t *testing.T) {
	t.Helper()
	confDir, exists := util.ConfDir, notebookPINBoxExists
	util.ConfDir = t.TempDir()
	notebookPINBoxExists = func(string) bool { return true }
	notebookPINLoaded, notebookPINUnlocked, notebookPINFailures = nil, map[string]time.Time{}, map[string]int{}
	t.Cleanup(func() {
		util.ConfDir, notebookPINBoxExists = confDir, exists
		notebookPINLoaded, notebookPINUnlocked, notebookPINFailures = nil, map[string]time.Time{}, map[string]int{}
	})
}

func TestNotebookPINLifecycle(t *testing.T) {
	resetNotebookPINForTest(t)
	box := "20261107120000-abcdefg"
	if IsNotebookPINLocked(box) || IsNotebookPINProtected(box) {
		t.Fatal("a notebook without PIN is not locked")
	}
	if err := SetNotebookPIN(box, "", "12"); !errors.Is(err, ErrNotebookPINFormat) {
		t.Fatalf("short PIN: %v", err)
	}
	if err := SetNotebookPIN(box, "", "482916"); err != nil {
		t.Fatal(err)
	}
	if IsNotebookPINLocked(box) {
		t.Fatal("setting a PIN keeps the open notebook unlocked")
	}
	notebookPINLock.Lock()
	delete(notebookPINUnlocked, box)
	notebookPINLock.Unlock()
	if !IsNotebookPINLocked(box) {
		t.Fatal("locked after LockNotebookPIN state")
	}
	if err := UnlockNotebookPIN(box, "000000"); !errors.Is(err, ErrNotebookPINWrong) {
		t.Fatalf("wrong PIN: %v", err)
	}
	if err := UnlockNotebookPIN(box, "482916"); err != nil || IsNotebookPINLocked(box) {
		t.Fatalf("unlock: %v", err)
	}

	// Idle longer than the auto-lock delay locks again; activity keeps it unlocked.
	notebookPINLock.Lock()
	notebookPINUnlocked[box] = time.Now().Add(-14 * time.Minute)
	notebookPINLock.Unlock()
	TouchNotebookPINs()
	if IsNotebookPINLocked(box) {
		t.Fatal("touch keeps it unlocked")
	}
	notebookPINLock.Lock()
	notebookPINUnlocked[box] = time.Now().Add(-16 * time.Minute)
	notebookPINLock.Unlock()
	if !IsNotebookPINLocked(box) {
		t.Fatal("idle for 16 minutes locks with the default 15")
	}
	if err := SetNotebookPINMinutes(0); err != nil || IsNotebookPINLocked(box) {
		t.Fatalf("0 minutes never locks: %v", err)
	}

	// The configuration survives a reload; changing the PIN needs the current one.
	notebookPINLock.Lock()
	notebookPINLoaded = nil
	notebookPINLock.Unlock()
	if !IsNotebookPINProtected(box) {
		t.Fatal("PIN saved")
	}
	if err := SetNotebookPIN(box, "1111", "246810"); !errors.Is(err, ErrNotebookPINWrong) {
		t.Fatalf("change with wrong current PIN: %v", err)
	}
	if err := SetNotebookPIN(box, "482916", "246810"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveNotebookPIN(box, "246810"); err != nil || IsNotebookPINProtected(box) {
		t.Fatalf("remove: %v", err)
	}
	statuses, minutes := GetNotebookPINStatus()
	if len(statuses) != 0 || minutes != 0 {
		t.Fatalf("status after removal: %v %d", statuses, minutes)
	}
}
