// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// revlis3m mods: PIN lock for chosen notebooks.
//
// A locked notebook is a closed notebook that only opens with its PIN. Closing it already removes it from the
// document tree, search, backlinks and embeds, and closes its tabs. This file guards the opening, and names what
// belongs to a locked notebook (its ID and the IDs of its databases) so that requests reaching for its files,
// history or databases by another way can be refused (api/mods_notebook_lock_guard.go). It is a privacy lock for
// people using the same SiYuan, not encryption: the files on disk are unchanged.
// The PINs are kept in the workspace configuration directory (conf/mods-notebook-lock.json), which is not synced.
//
// Guessing: PINs are checked one at a time. After notebookPINMaxFailures wrong PINs in a row the notebook refuses
// every PIN for 5 minutes, doubling at each new lockout (up to a day); this is saved, so a restart does not reset it.
// Relock: each notebook locks again after the idle delay without activity in that notebook (TouchNotebookPIN).

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
	notebookPINIterations     = 210000
	notebookPINDefaultMinutes = 15
	notebookPINMinLength      = 4
	notebookPINMaxLength      = 64
	notebookPINMaxFailures    = 5
	notebookPINLockoutBase    = 5 * time.Minute
	notebookPINLockoutMax     = 24 * time.Hour
)

var (
	ErrNotebookPINLocked  = errors.New("this notebook is locked with a PIN")
	ErrNotebookPINWrong   = errors.New("wrong PIN")
	ErrNotebookPINTooMany = errors.New("too many wrong PINs")
	ErrNotebookPINFormat  = errors.New("the PIN must have 4 to 64 characters")
	ErrNotebookPINNone    = errors.New("this notebook has no PIN")
)

type notebookPIN struct {
	Salt string `json:"salt"`
	Hash string `json:"hash"`
	// Failures counts wrong PINs since the last right one or the last lockout.
	Failures int `json:"failures,omitempty"`
	// Lockouts counts lockouts since the last right PIN; each one doubles the next wait.
	Lockouts int `json:"lockouts,omitempty"`
	// LockedUntil is the Unix time until which every PIN is refused.
	LockedUntil int64 `json:"lockedUntil,omitempty"`
}

type notebookPINConf struct {
	Minutes int                     `json:"minutes"`
	Boxes   map[string]*notebookPIN `json:"boxes"`
}

var (
	// notebookPINLock guards the state below. notebookPINVerifyLock makes PIN checks run one at a time; it is
	// taken before notebookPINLock, never the other way round.
	notebookPINLock       sync.Mutex
	notebookPINVerifyLock sync.Mutex
	notebookPINLoaded     *notebookPINConf
	notebookPINUnlocked   = map[string]time.Time{} // box ID -> last activity in that notebook
	notebookPINAVs        = map[string][]string{}  // box ID -> IDs of the databases in its documents, while locked
)

var (
	notebookPINBoxExists = func(boxID string) bool { return Conf.GetBox(boxID) != nil }
	notebookPINNow       = time.Now
)

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
	for id, entry := range conf.Boxes {
		if entry == nil {
			delete(conf.Boxes, id)
		}
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

func notebookPINTooManyError(until, now time.Time) error {
	minutes := int(until.Sub(now)/time.Minute) + 1
	return fmt.Errorf("%w, try again in %d min", ErrNotebookPINTooMany, minutes)
}

// notebookPINLockoutDuration is the wait after the n-th lockout in a row: 5 min, 10, 20… up to a day.
func notebookPINLockoutDuration(lockouts int) time.Duration {
	duration := notebookPINLockoutBase
	for i := 1; i < lockouts && duration < notebookPINLockoutMax; i++ {
		duration *= 2
	}
	return min(duration, notebookPINLockoutMax)
}

// verifyNotebookPIN checks a PIN. Checks run one at a time, so guesses sent together gain nothing, and after
// notebookPINMaxFailures wrong PINs in a row every PIN is refused for a while. Must be called WITHOUT notebookPINLock.
func verifyNotebookPIN(boxID, pin string) error {
	notebookPINVerifyLock.Lock()
	defer notebookPINVerifyLock.Unlock()

	notebookPINLock.Lock()
	entry := loadNotebookPINConf().Boxes[boxID]
	if entry == nil {
		notebookPINLock.Unlock()
		return ErrNotebookPINNone
	}
	now := notebookPINNow()
	if until := time.Unix(entry.LockedUntil, 0); entry.LockedUntil > 0 && now.Before(until) {
		notebookPINLock.Unlock()
		return notebookPINTooManyError(until, now)
	}
	saltHex, want := entry.Salt, entry.Hash
	notebookPINLock.Unlock()

	// The slow part runs outside notebookPINLock so that requests asking "is it locked?" are not held up.
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return err
	}
	hash, err := hashNotebookPIN(pin, salt)
	if err != nil {
		return err
	}
	right := subtle.ConstantTimeCompare([]byte(hash), []byte(want)) == 1

	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	conf := loadNotebookPINConf()
	entry = conf.Boxes[boxID]
	if entry == nil {
		return ErrNotebookPINNone
	}
	if !right {
		entry.Failures++
		var lockedOut error
		if entry.Failures >= notebookPINMaxFailures {
			entry.Failures = 0
			entry.Lockouts++
			until := now.Add(notebookPINLockoutDuration(entry.Lockouts))
			entry.LockedUntil = until.Unix()
			lockedOut = notebookPINTooManyError(until, now)
			logging.LogWarnf("notebook [%s]: too many wrong PINs, refusing PINs until %s", boxID, until.Format(time.RFC3339))
		}
		if err = saveNotebookPINConf(conf); err != nil {
			logging.LogErrorf("save notebook PIN conf failed: %s", err)
		}
		if lockedOut != nil {
			return lockedOut
		}
		return ErrNotebookPINWrong
	}
	if entry.Failures != 0 || entry.Lockouts != 0 || entry.LockedUntil != 0 {
		entry.Failures, entry.Lockouts, entry.LockedUntil = 0, 0, 0
		if err = saveNotebookPINConf(conf); err != nil {
			logging.LogErrorf("save notebook PIN conf failed: %s", err)
		}
	}
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
	return conf.Minutes > 0 && notebookPINNow().Sub(last) > time.Duration(conf.Minutes)*time.Minute
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
	if IsNotebookPINProtected(boxID) {
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
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	conf := loadNotebookPINConf()
	conf.Boxes[boxID] = &notebookPIN{Salt: hex.EncodeToString(salt), Hash: hash}
	// Setting a PIN on an open notebook keeps it open until the next lock.
	notebookPINUnlocked[boxID] = notebookPINNow()
	delete(notebookPINAVs, boxID)
	return saveNotebookPINConf(conf)
}

// RemoveNotebookPIN removes a notebook's PIN after checking it.
func RemoveNotebookPIN(boxID, pin string) error {
	if err := verifyNotebookPIN(boxID, pin); err != nil {
		return err
	}
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	conf := loadNotebookPINConf()
	delete(conf.Boxes, boxID)
	delete(notebookPINUnlocked, boxID)
	delete(notebookPINAVs, boxID)
	return saveNotebookPINConf(conf)
}

// UnlockNotebookPIN checks the PIN and allows the notebook to be opened until it is idle for the auto-lock delay.
func UnlockNotebookPIN(boxID, pin string) error {
	if err := verifyNotebookPIN(boxID, pin); err != nil {
		return err
	}
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	notebookPINUnlocked[boxID] = notebookPINNow()
	delete(notebookPINAVs, boxID)
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
	delete(notebookPINAVs, boxID)
	notebookPINLock.Unlock()
	if Conf.Box(boxID) != nil {
		Unmount(boxID)
	}
	return nil
}

// SetNotebookPINMinutes sets the idle minutes before unlocked notebooks lock again (0 = never). The delay is
// shared by all PIN notebooks; changing it needs the PIN of one of them, so that nobody can switch relocking off.
func SetNotebookPINMinutes(boxID, pin string, minutes int) error {
	if err := verifyNotebookPIN(boxID, pin); err != nil {
		return err
	}
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	conf := loadNotebookPINConf()
	conf.Minutes = max(minutes, 0)
	return saveNotebookPINConf(conf)
}

// TouchNotebookPIN records activity in one unlocked notebook, which postpones its relock. Activity elsewhere in
// the app does not count.
func TouchNotebookPIN(boxID string) {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	if _, unlocked := notebookPINUnlocked[boxID]; unlocked && !isNotebookPINLockedLocked(boxID) {
		notebookPINUnlocked[boxID] = notebookPINNow()
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

var notebookPINAVPattern = regexp.MustCompile(`"AttributeViewID":"([0-9]{14}-[0-9a-z]{7})"`)

// scanNotebookPINAVs lists the IDs of the databases placed in the notebook's documents, read from its files
// (they are readable while the notebook is closed: a PIN notebook is not encrypted).
func scanNotebookPINAVs(boxID string) []string {
	seen := map[string]bool{}
	var ret []string
	root := filepath.Join(util.DataDir, boxID)
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sy") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, match := range notebookPINAVPattern.FindAllSubmatch(data, -1) {
			if id := string(match[1]); !seen[id] {
				seen[id] = true
				ret = append(ret, id)
			}
		}
		return nil
	})
	sort.Strings(ret)
	return ret
}

// LockedNotebookPINTerms returns what must not be reachable right now: the ID of every locked PIN notebook and
// the IDs of the databases in its documents, each mapped to its notebook. It is empty when nothing is locked.
// A database placed in a locked notebook is hidden everywhere while that notebook is locked, mirrors included.
func LockedNotebookPINTerms() map[string]string {
	notebookPINLock.Lock()
	defer notebookPINLock.Unlock()
	var ret map[string]string
	for id := range loadNotebookPINConf().Boxes {
		if !isNotebookPINLockedLocked(id) {
			continue
		}
		if ret == nil {
			ret = map[string]string{}
		}
		ret[id] = id
		avs, scanned := notebookPINAVs[id]
		if !scanned {
			avs = scanNotebookPINAVs(id)
			notebookPINAVs[id] = avs
		}
		for _, avID := range avs {
			ret[avID] = id
		}
	}
	return ret
}

// notebookPINHistoryFilter is added to history searches so that a locked notebook's documents and databases
// do not show in the results. The terms are node IDs, so they are safe inside the statement.
func notebookPINHistoryFilter() (stmt string) {
	terms := LockedNotebookPINTerms()
	ids := make([]string, 0, len(terms))
	for term := range terms {
		if ast.IsNodeIDPattern(term) {
			ids = append(ids, term)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		stmt += " AND path NOT LIKE '%" + id + "%'"
	}
	return
}
