package tui

// getToolStatusIcon returns an appropriate icon based on tool execution status
func getToolStatusIcon(hasError bool) string {
	if hasError {
		return "❌"
	}
	return "✅"
}
