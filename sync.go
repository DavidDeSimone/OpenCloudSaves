package main

import (
	"path/filepath"
)

func SyncFilter(s string, syncDataPath Datapath) bool {
	for _, ignore := range syncDataPath.Ignore {
		if s == ignore {
			return true
		}
	}

	if len(syncDataPath.Exts) == 0 {
		return false
	}

	anyExtMatches := false
	fileExt := filepath.Ext(s)
	for _, ext := range syncDataPath.Exts {
		if ext == fileExt {
			anyExtMatches = true
			break
		}
	}

	return !anyExtMatches
}
