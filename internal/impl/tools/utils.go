package tools

import (
	"github.com/dustin/go-humanize"
)

type LineResult struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

func formatSize(size int64) string {
	return humanize.Bytes(uint64(size))
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
