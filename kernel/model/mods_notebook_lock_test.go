// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/siyuan-note/siyuan/kernel/util"
)

// resetNotebookPINForTest isolates the lock state and returns a function that moves the lock's clock forward.
func resetNotebookPINForTest(t *testing.T) (advance func(time.Duration)) {
	t.Helper()
	confDir, dataDir, exists, now := util.ConfDir, util.DataDir, notebookPINBoxExists, notebookPINNow
	util.ConfDir, util.DataDir = t.TempDir(), t.TempDir()
	notebookPINBoxExists = func(string) bool { return true }
	clock := time.Now()
	var clockLock sync.Mutex
	notebookPINNow = func() time.Time {
		clockLock.Lock()
		defer clockLock.Unlock()
		return clock
	}
	reset := func() {
		notebookPINLock.Lock()
		notebookPINLoaded, notebookPINUnlocked, notebookPINAVs = nil, map[string]time.Time{}, map[string][]string{}
		notebookPINLock.Unlock()
	}
	reset()
	t.Cleanup(func() {
		util.ConfDir, util.DataDir, notebookPINBoxExists, notebookPINNow = confDir, dataDir, exists, now
		reset()
	})
	return func(duration time.Duration) {
		clockLock.Lock()
		clock = clock.Add(duration)
		clockLock.Unlock()
	}
}

func lockNotebookPINForTest(box string) {
	notebookPINLock.Lock()
	delete(notebookPINUnlocked, box)
	delete(notebookPINAVs, box)
	notebookPINLock.Unlock()
}

func reloadNotebookPINForTest() {
	notebookPINLock.Lock()
	notebookPINLoaded = nil
	notebookPINLock.Unlock()
}

func TestNotebookPINLifecycle(t *testing.T) {
	advance := resetNotebookPINForTest(t)
	box, other := "20261107120000-abcdefg", "20261107120000-hijklmn"
	if IsNotebookPINLocked(box) || IsNotebookPINProtected(box) {
		t.Fatal("a notebook without PIN is not locked")
	}
	if err := SetNotebookPIN(box, "", "12"); !errors.Is(err, ErrNotebookPINFormat) {
		t.Fatalf("short PIN: %v", err)
	}
	if err := SetNotebookPIN(box, "", "482916"); err != nil {
		t.Fatal(err)
	}
	if err := SetNotebookPIN(other, "", "135790"); err != nil {
		t.Fatal(err)
	}
	if IsNotebookPINLocked(box) {
		t.Fatal("setting a PIN keeps the open notebook unlocked")
	}
	lockNotebookPINForTest(box)
	if !IsNotebookPINLocked(box) {
		t.Fatal("locked after LockNotebookPIN state")
	}
	if err := UnlockNotebookPIN(box, "000000"); !errors.Is(err, ErrNotebookPINWrong) {
		t.Fatalf("wrong PIN: %v", err)
	}
	if err := UnlockNotebookPIN(box, "482916"); err != nil || IsNotebookPINLocked(box) {
		t.Fatalf("unlock: %v", err)
	}

	// Relock is per notebook: activity in one notebook does not keep another one unlocked.
	advance(14 * time.Minute)
	TouchNotebookPIN(box)
	advance(2 * time.Minute)
	if IsNotebookPINLocked(box) {
		t.Fatal("activity in the notebook 2 minutes ago keeps it unlocked")
	}
	if !IsNotebookPINLocked(other) {
		t.Fatal("the other notebook had no activity for 16 minutes: it locks with the default 15")
	}
	advance(14 * time.Minute)
	if !IsNotebookPINLocked(box) {
		t.Fatal("idle for 16 minutes locks")
	}
	TouchNotebookPIN(box)
	if !IsNotebookPINLocked(box) {
		t.Fatal("activity does not unlock a locked notebook")
	}

	// Changing the delay needs a PIN.
	if err := SetNotebookPINMinutes(box, "000000", 0); !errors.Is(err, ErrNotebookPINWrong) {
		t.Fatalf("delay with a wrong PIN: %v", err)
	}
	if _, minutes := GetNotebookPINStatus(); minutes != notebookPINDefaultMinutes {
		t.Fatalf("delay changed without the PIN: %d", minutes)
	}
	if err := UnlockNotebookPIN(box, "482916"); err != nil {
		t.Fatal(err)
	}
	if err := SetNotebookPINMinutes(box, "482916", 0); err != nil {
		t.Fatalf("delay with the PIN: %v", err)
	}
	advance(48 * time.Hour)
	if IsNotebookPINLocked(box) {
		t.Fatal("0 minutes never locks")
	}

	// The configuration survives a reload; changing the PIN needs the current one.
	reloadNotebookPINForTest()
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
	if len(statuses) != 1 || statuses[0].Notebook != other || minutes != 0 {
		t.Fatalf("status after removal: %v %d", statuses, minutes)
	}
}

func TestNotebookPINLockout(t *testing.T) {
	advance := resetNotebookPINForTest(t)
	box := "20261107120000-abcdefg"
	if err := SetNotebookPIN(box, "", "482916"); err != nil {
		t.Fatal(err)
	}
	lockNotebookPINForTest(box)

	// Four wrong PINs are only wrong; the fifth starts a lockout.
	for i := 1; i < notebookPINMaxFailures; i++ {
		if err := UnlockNotebookPIN(box, "000000"); !errors.Is(err, ErrNotebookPINWrong) {
			t.Fatalf("wrong PIN %d: %v", i, err)
		}
	}
	if err := UnlockNotebookPIN(box, "000000"); !errors.Is(err, ErrNotebookPINTooMany) {
		t.Fatalf("fifth wrong PIN: %v", err)
	}
	// During the lockout even the right PIN is refused, and a restart does not clear it.
	if err := UnlockNotebookPIN(box, "482916"); !errors.Is(err, ErrNotebookPINTooMany) {
		t.Fatalf("right PIN during the lockout: %v", err)
	}
	reloadNotebookPINForTest()
	advance(4 * time.Minute)
	if err := UnlockNotebookPIN(box, "482916"); !errors.Is(err, ErrNotebookPINTooMany) || !IsNotebookPINLocked(box) {
		t.Fatalf("right PIN after a reload, 4 minutes in: %v", err)
	}
	if err := RemoveNotebookPIN(box, "482916"); !errors.Is(err, ErrNotebookPINTooMany) {
		t.Fatalf("removing the PIN during the lockout: %v", err)
	}

	// After 5 minutes PINs are checked again; five more wrong ones lock out for 10 minutes.
	advance(2 * time.Minute)
	for i := 1; i < notebookPINMaxFailures; i++ {
		if err := UnlockNotebookPIN(box, "000000"); !errors.Is(err, ErrNotebookPINWrong) {
			t.Fatalf("second round, wrong PIN %d: %v", i, err)
		}
	}
	if err := UnlockNotebookPIN(box, "000000"); !errors.Is(err, ErrNotebookPINTooMany) {
		t.Fatalf("second lockout: %v", err)
	}
	advance(6 * time.Minute)
	if err := UnlockNotebookPIN(box, "482916"); !errors.Is(err, ErrNotebookPINTooMany) {
		t.Fatalf("the second lockout lasts 10 minutes: %v", err)
	}
	advance(5 * time.Minute)
	if err := UnlockNotebookPIN(box, "482916"); err != nil || IsNotebookPINLocked(box) {
		t.Fatalf("right PIN after the lockout: %v", err)
	}

	// The right PIN clears the count: the next lockout starts again at 5 minutes after 5 wrong PINs.
	lockNotebookPINForTest(box)
	for i := 1; i < notebookPINMaxFailures; i++ {
		if err := UnlockNotebookPIN(box, "000000"); !errors.Is(err, ErrNotebookPINWrong) {
			t.Fatalf("after reset, wrong PIN %d: %v", i, err)
		}
	}
	if got := notebookPINLockoutDuration(1); got != 5*time.Minute {
		t.Fatalf("first lockout = %v", got)
	}
	if got := notebookPINLockoutDuration(40); got != notebookPINLockoutMax {
		t.Fatalf("lockouts are capped: %v", got)
	}
}

// Guesses sent at the same time gain nothing: only notebookPINMaxFailures of them are checked before the lockout,
// so the right PIN hidden among many wrong ones is not accepted.
func TestNotebookPINParallelGuesses(t *testing.T) {
	resetNotebookPINForTest(t)
	box := "20261107120000-abcdefg"
	if err := SetNotebookPIN(box, "", "482916"); err != nil {
		t.Fatal(err)
	}
	lockNotebookPINForTest(box)
	for i := 0; i < notebookPINMaxFailures; i++ {
		_ = UnlockNotebookPIN(box, "000000")
	}

	var wait sync.WaitGroup
	results := make([]error, 20)
	for i := range results {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			pin := "000000"
			if i == len(results)-1 {
				pin = "482916"
			}
			results[i] = UnlockNotebookPIN(box, pin)
		}(i)
	}
	wait.Wait()
	for i, err := range results {
		if !errors.Is(err, ErrNotebookPINTooMany) {
			t.Fatalf("guess %d during the lockout: %v", i, err)
		}
	}
	if !IsNotebookPINLocked(box) {
		t.Fatal("still locked")
	}
}

func TestLockedNotebookPINTerms(t *testing.T) {
	resetNotebookPINForTest(t)
	box, open := "20261107120000-abcdefg", "20261107120000-hijklmn"
	database, openDatabase := "20261107120100-avavava", "20261107120100-bvbvbvb"
	for notebook, avID := range map[string]string{box: database, open: openDatabase} {
		dir := filepath.Join(util.DataDir, notebook, "20261107120200-docdocd")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		doc := `{"ID":"20261107120300-subdocs","Type":"NodeDocument","Children":[{"Type":"NodeAttributeView","Properties":{"id":"20261107120400-blockid"},"AttributeViewID":"` + avID + `","AttributeViewType":"table"}]}`
		if err := os.WriteFile(filepath.Join(dir, "20261107120300-subdocs.sy"), []byte(doc), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if len(LockedNotebookPINTerms()) != 0 {
		t.Fatal("nothing is locked without a PIN")
	}
	if err := SetNotebookPIN(box, "", "482916"); err != nil {
		t.Fatal(err)
	}
	if len(LockedNotebookPINTerms()) != 0 {
		t.Fatal("an unlocked PIN notebook hides nothing")
	}
	lockNotebookPINForTest(box)
	terms := LockedNotebookPINTerms()
	if terms[box] != box || terms[database] != box || len(terms) != 2 {
		t.Fatalf("locked terms: %v", terms)
	}
	if filter := notebookPINHistoryFilter(); filter != " AND path NOT LIKE '%"+box+"%' AND path NOT LIKE '%"+database+"%'" {
		t.Fatalf("history filter: %q", filter)
	}
	if err := UnlockNotebookPIN(box, "482916"); err != nil || len(LockedNotebookPINTerms()) != 0 || notebookPINHistoryFilter() != "" {
		t.Fatalf("nothing is hidden once unlocked: %v", err)
	}
}
