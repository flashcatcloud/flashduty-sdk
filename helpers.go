package flashduty

import (
	"strings"
)

func parseCommaSeparatedStrings(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// mergeChannelIDs combines a primary slice with a deprecated singular ChannelID
// field. The slice wins when set; otherwise a non-zero singular value is wrapped
// into a one-element slice. Used to migrate callers from ChannelID to ChannelIDs
// without breaking existing code.
func mergeChannelIDs(channelIDs []int64, channelID int64) []int64 {
	if len(channelIDs) > 0 {
		return channelIDs
	}
	if channelID > 0 {
		return []int64{channelID}
	}
	return nil
}
