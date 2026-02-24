package formatters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// formatDiff formats diff content with colors and proper formatting
func FormatDiff(diff string) string {
	var diffContent string

	if strings.Contains(diff, "```diff") {
		// Extract diff content from markdown code block
		start := strings.Index(diff, "```diff\n")
		if start == -1 {
			diffContent = diff
		} else {
			start += 8 // Length of "```diff\n"
			end := strings.Index(diff[start:], "\n```")
			if end == -1 {
				diffContent = diff[start:]
			} else {
				// Extract the actual diff content (without the closing ```)
				diffContent = diff[start : start+end]
			}
		}
	} else {
		// Raw diff content
		diffContent = diff
	}

	// Check if this looks like a unified diff
	hasDiffMarkers := strings.Contains(diffContent, "---") ||
		strings.Contains(diffContent, "+++") ||
		strings.Contains(diffContent, "@@")

	if !hasDiffMarkers {
		// If it doesn't look like a diff, just return the content
		return diffContent
	}

	var output strings.Builder
	output.WriteString("Changes:\n")

	// Define styles for diff elements
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))     // Green
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))     // Red
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))    // Cyan
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))    // Blue

	lines := strings.Split(diffContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			output.WriteString(addStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			output.WriteString(delStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, "@@") {
			output.WriteString(hunkStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, " ") {
			output.WriteString(contextStyle.Render(line) + "\n")
		} else if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			output.WriteString(fileStyle.Render(line) + "\n")
		} else if line != "" {
			output.WriteString(line + "\n")
		}
	}

	return output.String()
}

// formatGenericResult tries to parse generic JSON results
func FormatGenericResult(result string) string {
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(result), &jsonData); err != nil {
		// If not JSON, check if it's a long text and truncate if needed
		if len(result) > 500 {
			lines := strings.Split(result, "\n")
			if len(lines) > 8 {
				var output strings.Builder
				for i := 0; i < 8; i++ {
					output.WriteString(lines[i] + "\n")
				}
				output.WriteString(fmt.Sprintf("... and %d more lines\n", len(lines)-8))
				return output.String()
			}
		}
		return result // Return raw if not JSON and not too long
	}

	var output strings.Builder
	for key, value := range jsonData {
		// Handle long string values
		if str, ok := value.(string); ok && len(str) > 200 {
			lines := strings.Split(str, "\n")
			if len(lines) > 8 {
				var truncated strings.Builder
				for i := 0; i < 8; i++ {
					truncated.WriteString(lines[i] + "\n")
				}
				truncated.WriteString(fmt.Sprintf("... and %d more lines", len(lines)-8))
				value = truncated.String()
			}
		}
		output.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}

	return output.String()
}

// formatTokenCount formats a token count with comma separators for thousands
func FormatTokenCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}

	// Convert to string and add commas
	str := fmt.Sprintf("%d", count)
	var result strings.Builder

	// Process from right to left, adding commas every 3 digits
	for i, digit := range reversedString(str) {
		if i > 0 && i%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteByte(byte(digit))
	}

	return reversedString(result.String())
}

// reversedString returns the reverse of a string
func reversedString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
