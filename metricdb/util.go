package metricdb

import (
	"fmt"
	"sort"
	"strings"
)

// Prepare list of sorted unique tags joined by #
func normalizeTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	if len(tags) == 1 {
		return fmt.Sprintf("#%s#", tags[0])
	}

	sort.Strings(tags)
	tags = unique(tags)

	return fmt.Sprintf("#%s#", strings.Join(tags, "#"))
}

// Return slice with only unique elements. Slice must be sorted.
func unique(sortedTags []string) []string {
	if len(sortedTags) < 2 {
		return sortedTags
	}

	j := 0
	for i := 0; i < len(sortedTags); {
		for i < len(sortedTags) && sortedTags[i] == sortedTags[j] {
			i++
		}

		if i < len(sortedTags) {
			j++;
			if i != j {
				sortedTags[j] = sortedTags[i]
			}
		}
	}

	j++

	return sortedTags[:j]
}
