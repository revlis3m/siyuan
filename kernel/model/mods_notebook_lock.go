// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// revlis3m mods: PIN lock for chosen notebooks.
//
// A locked notebook is a closed notebook that only opens with its PIN. Closing it already removes it from the
// document tree, search, backlinks and embeds, and closes its tabs; this file only guards the opening. It is a
// privacy lock for people using the same SiYuan, not encryption: the files on disk are unchanged.
// The PINs are kept in the workspace configuration directory (conf/mods-notebook-lock.json), which is not synced.

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/88250/gulu"
	"github.com/88250/lute/ast"
	"github.com/siyuan-note/filelock"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/util"
)

const (
	notebookPINIterations      = 210000
	notebookPINDefaultMinutes  = 15
	notebookPINMinLength       = 4
	notebookPINMaxLength       = 64
	notebookPINMaxFailureDelay = 30 * time.Second
)

var (
	ErrNotebookPINLocked = errors.New("this notebook is locked with a PIN")
	ErrNotebookPINWrong  = errors.New("wrong PIN")
	ErrNotebookPINFormat = errors.New("the PIN must have 4 to 64 characters")
	ErrNotebookPINNone   = errors.New("this notebook has no PIN")
)

type notebookPIN struct {
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

type notebookPINConf struct {
	Minutes int                     `json:"minutes"`
	Boxes   map[string]*notebookPIN `json:"boxes"`
}

var (
	notebookPINLock     sync.Mutex
	notebookPINLoaded   *notebookPINConf
	notebookPINUnlocked = map[string]time.Time{} // box ID -> last activity
	notebookPINFailures = map[string]int{}
)

var notebookPINBoxExists = func(boxID string) bool { return Conf.GetBox(boxID) != nil }

func notebookPINConfPath() string {
	return filepath.Join(util.ConfDir, "mods-notebook-lock.json")
}

// loadNotebookPINConf must be called with notebookPINLock held.
func loadNotebookPINConf() *notebookPINConf {
	if notebookPINLoaded != nil {
		return notebookPINLoaded
	}
	conf := &notebookPINConf{Minutes: notebookPINDefaultMinutes, Boxes: map[string]*notebookPIN{}}
	if data, err := filelock.ReadFile(notebookPINConfPath()); err == nil {
		if err = gulu.JSON.UnmarshalJSON(data, conf); err != nil {
			logging.LogErrorf("read notebook PIN conf failed: %s", err)
		}
	} else if !os.IsNotExist(err) {
		logging.LogErrorf("read notebook PIN conf failed: %s", err)
	}
	if conf.Boxes == nil {
		conf.Boxes = map[string]*notebookPIN{}
	}
	if conf.Minutes < 0 {
		conf.Minutes = notebookPINDefaultMinutes
	}
	notebookPINLoaded = conf
	return conf
}

// saveNotebookPINConf must be called with notebookPINLock held.
func saveNotebookPINConf(conf *notebookPINConf) error {
	data, err := gulu.JSON.MarshalIndentJSON(conf, "", "  ")
	if err != nil {
		return err
	}
	return filelock.WriteFile(notebookPINConfPath(), data)
}

func hashNotebookPIN(pin string, salt []byte) (string, error) {
	key, err := pbkdf2.Key(sha256.New, pin, salt, notebookPINIterations, 32)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func checkNotebookPINFormat(pin string) error {
	if n := utf8.RuneCountInString(pin); n < notebookPINMinLength || n > notebookPINMaxLength {
		return ErrNotebookPINFormat
	}
	return nil
}

// verifyNotebookPIN checks a PIN and slows down repeated failures. Must be called with notebookPINLock held;
// it releases the lock while waiting.
func verifyNotebookPIN(boxID, pin string) error {
	entry := loadNotebookPINConf().Boxes[boxID]
	if entry == nil {
		return ErrNotebookPINNone
	}
	if failures := notebookPINFailures[boxID]; failures > 0 {
		delay := min(time.Duration(1<<min(failures-1, 5))*time.Second, notebookPINMaxFailureDelay)
		notebookPINLock.Unlock()
		time.Sleep(delay)
		notebookPINLock.Lock()
	}
	salt, err := hex.DecodeString(entry.Salt)
	if err != nil {
		return err
	}
	hash, err := hashNotebookPIN(pin, salt)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(hash), []byte(entry.Hash)) != 1 {
		notebookPINFailures[boxID]++
		return ErrNotebookPINWrong
	}
	delete(notebookPINFailures, boxID)
	return nil
}

// IsNotebookPINProtected reports whether the notebook has a PIN.
func IsNotebookPINProtected(boxID string) bool {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	return loadNotebookPINConf().Boxes[boxID] != nil
}

// IsNotebookPINLocked reports whether the notebook has a PIN and is not unlocked right now.
func IsNotebookPINLocked(boxID string) bool {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	return isNotebookPINLockedLocked(boxID)
}

func isNotebookPINLockedLocked(boxID string) bool {
	conf := loadNotebookPINConf()
	if conf.Boxes[boxID] == nil {
		return false
	}
	last, ok := notebookPINUnlocked[boxID]
	if !ok {
		return true
	}
	return conf.Minutes > 0 && time.Since(last) > time.Duration(conf.Minutes)*time.Minute
}

type NotebookPINStatus struct {
	Notebook string `json:"notebook"`
	Locked   bool   `json:"locked"`
}

// GetNotebookPINStatus lists the notebooks with a PIN and the auto-lock delay in minutes (0 = never).
func GetNotebookPINStatus() (boxes []*NotebookPINStatus, minutes int) {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	conf := loadNotebookPINConf()
	for id := range conf.Boxes {
		boxes = append(boxes, &NotebookPINStatus{Notebook: id, Locked: isNotebookPINLockedLocked(id)})
	}
	sort.Slice(boxes, func(i, j int) bool { return boxes[i].Notebook < boxes[j].Notebook })
	return boxes, conf.Minutes
}

// SetNotebookPIN sets or changes a notebook's PIN. Changing it needs the current PIN.
func SetNotebookPIN(boxID, currentPIN, pin string) error {
	if !ast.IsNodeIDPattern(boxID) || !notebookPINBoxExists(boxID) {
		return errors.New("notebook not found")
	}
	if err := checkNotebookPINFormat(pin); err != nil {
		return err
	}
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	conf := loadNotebookPINConf()
	if conf.Boxes[boxID] != nil {
		if err := verifyNotebookPIN(boxID, currentPIN); err != nil {
			return err
		}
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	hash, err := hashNotebookPIN(pin, salt)
	if err != nil {
		return err
	}
	conf.Boxes[boxID] = &notebookPIN{Salt: hex.EncodeToString(salt), Hash: hash}
	// Setting a PIN on an open notebook keeps it open until the next lock.
	notebookPINUnlocked[boxID] = time.Now()
	return saveNotebookPINConf(conf)
}

// RemoveNotebookPIN removes a notebook's PIN after checking it.
func RemoveNotebookPIN(boxID, pin string) error {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	if err := verifyNotebookPIN(boxID, pin); err != nil {
		return err
	}
	conf := loadNotebookPINConf()
	delete(conf.Boxes, boxID)
	delete(notebookPINUnlocked, boxID)
	return saveNotebookPINConf(conf)
}

// UnlockNotebookPIN checks the PIN and allows the notebook to be opened until it is idle for the auto-lock delay.
func UnlockNotebookPIN(boxID, pin string) error {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	if err := verifyNotebookPIN(boxID, pin); err != nil {
		return err
	}
	notebookPINUnlocked[boxID] = time.Now()
	return nil
}

// LockNotebookPIN locks a notebook now: it is closed and needs the PIN to open again.
func LockNotebookPIN(boxID string) error {
	notebookPINLock.Lock()
	if loadNotebookPINConf().Boxes[boxID] == nil {
		notebookPINLock.Unlock()
		return ErrNotebookPINNone
	}
	delete(notebookPINUnlocked, boxID)
	notebookPINLock.Unlock()
	if Conf.Box(boxID) != nil {
		Unmount(boxID)
	}
	return nil
}

// SetNotebookPINMinutes sets the idle minutes before unlocked notebooks lock again (0 = never).
func SetNotebookPINMinutes(minutes int) error {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	conf := loadNotebookPINConf()
	conf.Minutes = max(minutes, 0)
	return saveNotebookPINConf(conf)
}

// TouchNotebookPINs keeps unlocked notebooks unlocked while someone is using the app.
func TouchNotebookPINs() {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	now := time.Now()
	for id := range notebookPINUnlocked {
		if !isNotebookPINLockedLocked(id) {
			notebookPINUnlocked[id] = now
		}
	}
}

// AutoLockNotebookPINsJob closes PIN notebooks that are open without being unlocked (after a restart) or idle.
func AutoLockNotebookPINsJob() {
	notebookPINLock.Lock()
	var lock []string
	for id := range loadNotebookPINConf().Boxes {
		if isNotebookPINLockedLocked(id) {
			delete(notebookPINUnlocked, id)
			if Conf.Box(id) != nil {
				lock = append(lock, id)
			}
		}
	}
	notebookPINLock.Unlock()
	for _, id := range lock {
		logging.LogInfof("locking notebook [%s] (PIN)", id)
		Unmount(id)
	}
}
