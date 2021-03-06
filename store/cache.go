// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/osutil"
)

// downloadCache is the interface that a store download cache must provide
type downloadCache interface {
	// Get gets the given cacheKey content and puts it into targetPath
	Get(cacheKey, targetPath string) error
	// Put adds a new file to the cache
	Put(cacheKey, sourcePath string) error
}

// nullCache is cache that does not cache
type nullCache struct{}

func (cm *nullCache) Get(cacheKey, targetPath string) error {
	return fmt.Errorf("cannot get items from the nullCache")
}
func (cm *nullCache) Put(cacheKey, sourcePath string) error { return nil }

// changesByReverseMtime sorts by the mtime of files
type changesByReverseMtime []os.FileInfo

func (s changesByReverseMtime) Len() int           { return len(s) }
func (s changesByReverseMtime) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s changesByReverseMtime) Less(i, j int) bool { return s[i].ModTime().After(s[j].ModTime()) }

// cacheManager implements a downloadCache via content based hard linking
type CacheManager struct {
	cacheDir string
	maxItems int
}

// NewCacheManager returns a new CacheManager with the given cacheDir
// and the given maximum amount of items. The idea behind it is the
// following algorithm:
//
// 1. When starting a download, check if it exists in $cacheDir
// 2. If found, update its mtime, hardlink into target location, and
//    return success
// 3. If not found, download the snap
// 4. On success, hardlink into $cacheDir/<digest>
// 5. If cache dir has more than maxItems entries, remove oldest mtimes
//    until it has maxItems
//
// The caching part is done here, the downloading happens in the store.go
// code.
func NewCacheManager(cacheDir string, maxItems int) *CacheManager {
	return &CacheManager{
		cacheDir: cacheDir,
		maxItems: maxItems,
	}
}

// Get gets the given cacheKey content and puts it into targetPath
func (cm *CacheManager) Get(cacheKey, targetPath string) error {
	if err := os.Link(cm.path(cacheKey), targetPath); err != nil {
		return err
	}
	logger.Debugf("using cache for %s", targetPath)
	now := time.Now()
	return os.Chtimes(targetPath, now, now)
}

// Put adds a new file to the cache with the given cacheKey
func (cm *CacheManager) Put(cacheKey, sourcePath string) error {
	// always try to create the cache dir first or the following
	// osutil.IsWritable will always fail if the dir is missing
	_ = os.MkdirAll(cm.cacheDir, 0700)

	// happens on e.g. `snap download` which runs as the user
	if !osutil.IsWritable(cm.cacheDir) {
		return nil
	}

	err := os.Link(sourcePath, cm.path(cacheKey))
	if os.IsExist(err) {
		now := time.Now()
		err := os.Chtimes(cm.path(cacheKey), now, now)
		// this can happen if a cleanup happens in parallel, ie.
		// the file was there but cleanup() removed it between
		// the os.Link/os.Chtimes - no biggie, just link it again
		if os.IsNotExist(err) {
			return os.Link(sourcePath, cm.path(cacheKey))
		}
		return err
	}
	if err != nil {
		return err
	}
	return cm.cleanup()
}

// Count returns the number of items in the cache
func (cm *CacheManager) Count() int {
	if l, err := ioutil.ReadDir(cm.cacheDir); err == nil {
		return len(l)
	}
	return 0
}

// path returns the full path of the given content in the cache
func (cm *CacheManager) path(cacheKey string) string {
	return filepath.Join(cm.cacheDir, cacheKey)
}

// cleanup ensures that only maxItems are stored in the cache
func (cm *CacheManager) cleanup() error {
	fil, err := ioutil.ReadDir(cm.cacheDir)
	if err != nil {
		return err
	}
	if len(fil) <= cm.maxItems {
		return nil
	}

	sort.Sort(changesByReverseMtime(fil))
	for _, fi := range fil[cm.maxItems:] {
		if err := os.Remove(cm.path(fi.Name())); err != nil && os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
