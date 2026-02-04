package ui

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"strings"
)

// formatToolName formats tool names with relevant arguments for display
func formatToolName(toolName, arguments string) string {
	name, suffix := formatToolNameParts(toolName, arguments)
	displayName := name + ":"
	if suffix != "" {
		displayName += " " + suffix
	}
	return displayName
}

// formatToolNameParts returns name and suffix separately for templates
func formatToolNameParts(toolName, arguments string) (string, string) {
	switch toolName {
	case "FileRead", "FileWrite", "Directory":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.Path != "" {
			return toolName, args.Path
		}
	case "FileSearch":
		var args struct {
			Pattern   string `json:"pattern"`
			Directory string `json:"directory"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err == nil {
			detail := args.Pattern
			if args.Directory != "" && args.Directory != "." {
				detail += " in " + args.Directory
			}
			return toolName, detail
		}
	case "Process":
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.Command != "" {
			// Truncate long commands
			if len(args.Command) > 50 {
				args.Command = args.Command[:47] + "..."
			}
			return toolName, args.Command
		}
	}
	return toolName, ""
}

// formatToolResult formats tool execution results for Web UI display
func formatToolResult(toolName, result string, diff string, arguments string) template.HTML {
	switch toolName {
	case "FileWrite":
		return formatFileWriteResult(result, diff)
	case "FileRead":
		return formatFileReadResult(result)
	case "FileSearch":
		return formatFileSearchResult(result)
	case "Directory":
		return formatDirectoryResult(result)
	case "Process":
		return formatProcessResult(result, arguments)
	case "Project":
		return formatProjectResult(result)
	case "Memory":
		return formatMemoryResult(result)
	case "WebSearch":
		return formatWebSearchResult(result, arguments)
	default:
		// Try to extract summary from JSON responses
		var jsonResponse struct {
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal([]byte(result), &jsonResponse); err == nil && jsonResponse.Summary != "" {
			return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(jsonResponse.Summary)))
		}
		// For non-JSON results or JSON without summary, return truncated result
		return formatGenericResult(result)
	}
}

// formatFileWriteResult formats FileWrite tool results
func formatFileWriteResult(result string, diff string) template.HTML {
	var resultData struct {
		Summary     string `json:"summary"`
		Success     bool   `json:"success"`
		Path        string `json:"path"`
		Occurrences int    `json:"occurrences"`
		ReplacedAll bool   `json:"replaced_all"`
		Diff        string `json:"diff"`
	}

	var output strings.Builder

	// First, try to use the diff parameter if available
	if diff != "" {
		// Try to parse JSON to get summary
		if err := json.Unmarshal([]byte(result), &resultData); err == nil && resultData.Summary != "" {
			output.WriteString(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(resultData.Summary)))
			output.WriteString(string(formatDiff(diff)))
			return template.HTML(output.String())
		} else {
			// If JSON parsing fails, create a simple summary
			output.WriteString("<div class=\"tool-summary\">File modified successfully</div>")
			output.WriteString(string(formatDiff(diff)))
			return template.HTML(output.String())
		}
	}

	// If no diff parameter, try to parse the full JSON result
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		// If parsing fails, try to extract summary from JSON
		var jsonResponse struct {
			Summary string `json:"summary"`
		}
		if err2 := json.Unmarshal([]byte(result), &jsonResponse); err2 == nil && jsonResponse.Summary != "" {
			return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(jsonResponse.Summary)))
		}
		return formatGenericResult(result) // Return truncated if parsing fails
	}

	// Use the summary from the JSON response
	output.WriteString(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(resultData.Summary)))

	// Add the diff from JSON if available
	if resultData.Diff != "" {
		output.WriteString(string(formatDiff(resultData.Diff)))
	}

	return template.HTML(output.String())
}

// formatFileReadResult formats FileRead tool results
func formatFileReadResult(result string) template.HTML {
	var response struct {
		Summary string `json:"summary"`
		Path    string `json:"path"`
		Content string `json:"content"`
		Lines   int    `json:"lines"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return formatGenericResult(result)
	}

	// For Web UI, show simple summary without content preview
	if response.Lines > 0 {
		fileName := "file"
		if response.Path != "" {
			fileName = filepath.Base(response.Path)
		}
		return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">📄 %s (%d lines read)</div>", html.EscapeString(fileName), response.Lines))
	}

	if response.Content != "" {
		lines := strings.Split(response.Content, "\n")
		fileName := "file"
		if response.Path != "" {
			fileName = filepath.Base(response.Path)
		}
		return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">📄 %s (%d lines read)</div>", html.EscapeString(fileName), len(lines)))
	}

	return template.HTML("<div class=\"tool-summary\">File read successfully</div>")
}

// formatFileSearchResult formats FileSearch tool results
func formatFileSearchResult(result string) template.HTML {
	var response struct {
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return formatGenericResult(result)
	}

	return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(response.Summary)))
}

// formatDirectoryResult formats Directory tool results
func formatDirectoryResult(result string) template.HTML {
	var response struct {
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return formatGenericResult(result)
	}

	return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(response.Summary)))
}

// formatProcessResult formats Process tool results
func formatProcessResult(result string, arguments string) template.HTML {
	// First, try to parse as summary format (if tool returns {"summary": "..."})
	var summaryResponse struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(result), &summaryResponse); err == nil && summaryResponse.Summary != "" {
		// Fix incomplete summaries
		summary := summaryResponse.Summary
		if summary == "Executed: " {
			summary = "Executed successfully"
		}
		return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(summary)))
	}

	// Otherwise, parse as output format
	var response struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exit_code"`
		Error    string `json:"error"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return formatGenericResult(result)
	}

	var summary string
	if response.Error != "" {
		summary = fmt.Sprintf("Failed: %s", response.Error)
	} else if response.ExitCode != 0 {
		summary = fmt.Sprintf("Failed with exit code %d", response.ExitCode)
	} else {
		summary = "Executed successfully"
	}

	return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(summary)))
}

// formatWebSearchResult formats WebSearch tool results
func formatWebSearchResult(result string, arguments string) template.HTML {
	var args struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.Query != "" {
		return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">Searched for: %s</div>", html.EscapeString(args.Query)))
	}

	// Fallback
	var jsonResponse struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(result), &jsonResponse); err == nil && jsonResponse.Summary != "" {
		return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(jsonResponse.Summary)))
	}
	return template.HTML("<div class=\"tool-summary\">Web search completed</div>")
}

// formatProjectResult formats Project tool results
func formatProjectResult(result string) template.HTML {
	var response struct {
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return formatGenericResult(result)
	}

	return template.HTML(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(response.Summary)))
}

// formatMemoryResult formats Memory tool results
func formatMemoryResult(result string) template.HTML {
	var output strings.Builder

	// Try parsing as entities array
	var entities []interface{}
	if err := json.Unmarshal([]byte(result), &entities); err == nil && len(entities) > 0 {
		output.WriteString(fmt.Sprintf("<div class=\"tool-summary\">Memory Entities (%d created):</div>", len(entities)))
		output.WriteString("<ul class=\"memory-list\">")

		maxEntities := 5
		for i, entity := range entities {
			if i >= maxEntities {
				break
			}

			if entityMap, ok := entity.(map[string]interface{}); ok {
				name := entityMap["name"]
				entityType := entityMap["type"]
				output.WriteString(fmt.Sprintf("<li>%s (%s)</li>", html.EscapeString(fmt.Sprintf("%v", name)), html.EscapeString(fmt.Sprintf("%v", entityType))))
			}
		}

		if len(entities) > maxEntities {
			output.WriteString(fmt.Sprintf("<li>... and %d more entities</li>", len(entities)-maxEntities))
		}

		output.WriteString("</ul>")
		return template.HTML(output.String())
	}

	// Try parsing as relations array
	var relations []interface{}
	if err := json.Unmarshal([]byte(result), &relations); err == nil && len(relations) > 0 {
		output.WriteString(fmt.Sprintf("<div class=\"tool-summary\">Memory Relations (%d created):</div>", len(relations)))
		output.WriteString("<ul class=\"memory-list\">")

		maxRelations := 5
		for i, relation := range relations {
			if i >= maxRelations {
				break
			}

			if relationMap, ok := relation.(map[string]interface{}); ok {
				source := relationMap["source"]
				relationType := relationMap["type"]
				target := relationMap["target"]
				output.WriteString(fmt.Sprintf("<li>%s --%s--> %s</li>", html.EscapeString(fmt.Sprintf("%v", source)), html.EscapeString(fmt.Sprintf("%v", relationType)), html.EscapeString(fmt.Sprintf("%v", target))))
			}
		}

		if len(relations) > maxRelations {
			output.WriteString(fmt.Sprintf("<li>... and %d more relations</li>", len(relations)-maxRelations))
		}

		output.WriteString("</ul>")
		return template.HTML(output.String())
	}

	// Try parsing as graph structure
	var graph map[string]interface{}
	if err := json.Unmarshal([]byte(result), &graph); err == nil {
		if entities, ok := graph["entities"].([]interface{}); ok {
			output.WriteString(fmt.Sprintf("<div class=\"tool-summary\">Knowledge Graph - Entities (%d):</div>", len(entities)))
			output.WriteString("<ul class=\"memory-list\">")

			maxEntities := 5
			for i, entity := range entities {
				if i >= maxEntities {
					break
				}

				if entityMap, ok := entity.(map[string]interface{}); ok {
					name := entityMap["name"]
					entityType := entityMap["type"]
					output.WriteString(fmt.Sprintf("<li>%s (%s)</li>", html.EscapeString(fmt.Sprintf("%v", name)), html.EscapeString(fmt.Sprintf("%v", entityType))))
				}
			}

			if len(entities) > maxEntities {
				output.WriteString(fmt.Sprintf("<li>... and %d more entities</li>", len(entities)-maxEntities))
			}
			output.WriteString("</ul>")
		}

		if relations, ok := graph["relations"].([]interface{}); ok {
			output.WriteString(fmt.Sprintf("<div class=\"tool-summary\">Relations (%d):</div>", len(relations)))
			output.WriteString("<ul class=\"memory-list\">")

			maxRelations := 5
			for i, relation := range relations {
				if i >= maxRelations {
					break
				}

				if relationMap, ok := relation.(map[string]interface{}); ok {
					source := relationMap["source"]
					relationType := relationMap["type"]
					target := relationMap["target"]
					output.WriteString(fmt.Sprintf("<li>%s --%s--> %s</li>", html.EscapeString(fmt.Sprintf("%v", source)), html.EscapeString(fmt.Sprintf("%v", relationType)), html.EscapeString(fmt.Sprintf("%v", target))))
				}
			}

			if len(relations) > maxRelations {
				output.WriteString(fmt.Sprintf("<li>... and %d more relations</li>", len(relations)-maxRelations))
			}
			output.WriteString("</ul>")
		}

		if output.Len() > 0 {
			return template.HTML(output.String())
		}
	}

	return formatGenericResult(result)
}

// formatGenericResult tries to parse generic JSON results and limit output
func formatGenericResult(result string) template.HTML {
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &jsonData); err != nil {
		// If not JSON, check if it's a long text and truncate if needed
		if len(result) > 500 {
			lines := strings.Split(result, "\n")
			if len(lines) > 5 {
				var output strings.Builder
				for i := 0; i < 5; i++ {
					output.WriteString(html.EscapeString(lines[i]) + "\n")
				}
				output.WriteString(fmt.Sprintf("... and %d more lines", len(lines)-5))
				return template.HTML(fmt.Sprintf("<div class=\"tool-result-truncated\"><pre>%s</pre></div>", output.String()))
			}
		}
		return template.HTML(fmt.Sprintf("<div class=\"tool-result\"><pre>%s</pre></div>", html.EscapeString(result)))
	}

	var output strings.Builder
	output.WriteString("<div class=\"tool-result-json\">")
	for key, value := range jsonData {
		// Handle long string values
		if str, ok := value.(string); ok && len(str) > 200 {
			lines := strings.Split(str, "\n")
			if len(lines) > 5 {
				var truncated strings.Builder
				for i := 0; i < 5; i++ {
					truncated.WriteString(html.EscapeString(lines[i]) + "\n")
				}
				truncated.WriteString(fmt.Sprintf("... and %d more lines", len(lines)-5))
				value = truncated.String()
			}
		}
		output.WriteString(fmt.Sprintf("<div><strong>%s:</strong> %v</div>", html.EscapeString(key), value))
	}
	output.WriteString("</div>")

	return template.HTML(output.String())
}

// formatDiff formats diff content with HTML styling for Web UI
func formatDiff(diff string) template.HTML {
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
		return template.HTML(fmt.Sprintf("<div class=\"diff-content\"><pre>%s</pre></div>", html.EscapeString(diffContent)))
	}

	var output strings.Builder
	output.WriteString("<div class=\"diff-container\">")
	output.WriteString("<div class=\"diff-header\">Changes:</div>")
	output.WriteString("<div class=\"diff-content\">")

	lines := strings.Split(diffContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			output.WriteString(fmt.Sprintf("<div class=\"diff-line diff-add\">%s</div>", html.EscapeString(line)))
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			output.WriteString(fmt.Sprintf("<div class=\"diff-line diff-del\">%s</div>", html.EscapeString(line)))
		} else if strings.HasPrefix(line, "@@") {
			output.WriteString(fmt.Sprintf("<div class=\"diff-line diff-hunk\">%s</div>", html.EscapeString(line)))
		} else if strings.HasPrefix(line, " ") {
			output.WriteString(fmt.Sprintf("<div class=\"diff-line diff-context\">%s</div>", html.EscapeString(line)))
		} else if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			output.WriteString(fmt.Sprintf("<div class=\"diff-line diff-file\">%s</div>", html.EscapeString(line)))
		} else if line != "" {
			output.WriteString(fmt.Sprintf("<div class=\"diff-line\">%s</div>", html.EscapeString(line)))
		}
	}

	output.WriteString("</div>")
	output.WriteString("</div>")

	return template.HTML(output.String())
}
