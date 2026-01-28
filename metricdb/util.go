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

// Check if string matches tags stored in storage
func matchTags(tags []string, storageTags string) bool {
	if len(tags) == 0 {
		return true
	}

	st := make(map[string]bool)
	for _, t := range strings.Split(storageTags, "#") {
		if t == "" {
			continue
		}
		st[t] = true
	}

	for _, t := range tags {
		r, ok := st[t]
		if !ok || !r {
			return false
		}
	}

	return true
}

// Split name and tags from name#t1#t2 form
func splitNameTags(nmt string) (string, []string) {
	res := strings.Split(nmt, "#")
	return res[0], res[1:]
}

// Split tags and return tag array
func splitTagsString(tags string) []string {
	res := strings.Split(tags, "#")
	l := 0
	for ; l < len(res); l++ {
		if res[l] != "" {
			break
		}
	}

	r := l
	for ; r < len(res); r++ {
		if res[r] == "" {
			break
		}
	}

	return res[l: r]
}
