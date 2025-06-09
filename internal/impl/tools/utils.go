package tools

import (
	"github.com/dustin/go-humanize"
)

func formatSize(size int64) string {
	return humanize.Bytes(uint64(size))
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
