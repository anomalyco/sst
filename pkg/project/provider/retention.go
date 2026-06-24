package provider

import (
	"encoding/hex"
	"path"
	"sort"
	"strings"

	"github.com/sst/sst/v3/pkg/id"
)

var historyKinds = []string{"snapshot", "eventlog", "update", "summary"}

func staleUpdateIDs(keys []string, retention int) []string {
	updateIDs := make([]string, 0, len(keys))
	seen := map[string]struct{}{}
	for _, key := range keys {
		updateID, ok := updateIDFromKey(key)
		if !ok {
			continue
		}
		if _, ok := seen[updateID]; ok {
			continue
		}
		seen[updateID] = struct{}{}
		updateIDs = append(updateIDs, updateID)
	}

	// Descending IDs encode newer timestamps as lexicographically smaller values.
	sort.Strings(updateIDs)
	if len(updateIDs) <= retention {
		return nil
	}
	return updateIDs[retention:]
}

func updateIDFromKey(key string) (string, bool) {
	updateID := strings.TrimSuffix(path.Base(key), ".json")
	if len(updateID) != id.LENGTH {
		return "", false
	}
	if _, err := hex.DecodeString(updateID); err != nil {
		return "", false
	}
	return updateID, true
}
