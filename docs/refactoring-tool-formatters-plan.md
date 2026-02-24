# Refactoring Plan for Tool Formatters in Chat View and WebUI

## Overview
This plan outlines the refactoring of tool formatting logic in both the TUI (`chat_view.go`) and WebUI (`internal/ui/formatters.go`) to implement a strategy pattern. Currently, both UIs use large switch statements in `formatToolName` and `formatToolResult` functions, which violate the open-closed principle. By moving formatting logic into the tools themselves, adding a new tool will only require implementing the `Tool` interface methods.

## Current State Analysis
- **TUI (`internal/tui/chat_view.go`)**: Contains ~25 formatting functions with tool-specific logic using switch statements on tool names. Depends on result JSON, arguments, and diff data.
- **WebUI (`internal/ui/formatters.go`)**: Similar structure with HTML-based formatting (returning `template.HTML`) for web display.
- **Tool Interface**: Defined in `internal/domain/entities/tool.go` with basic methods (Name, Description, etc.).
- **Tool Implementations**: 22 tools in `internal/impl/tools/` that implement the `Tool` interface.
- **Shared Issues**: Both UIs have nearly identical logic for parsing JSON results, handling arguments, and applying diffs, but output formats differ (text vs. HTML).

## Proposed Changes
1. **Create Shared Packages**: Move common utilities (e.g., JSON parsing, diff handling) to `internal/tui/formatters/` (TUI) and `internal/ui/formatters/` (WebUI), adapting for each UI's output format.
2. **Extend Tool Interface**: Add `FormatResult(ui string, result, diff, arguments string) string` and `DisplayName(ui string, arguments string) (string, string)` methods. The `ui` parameter ("tui" or "webui") determines output format.
3. **Implement in Tools**: Each tool implements UI-aware formatting, using shared utilities.
4. **Update UIs**: Replace switch statements with calls to tool methods.
5. **Cleanup**: Remove old formatter functions.

## Benefits
- Adding a new tool only requires implementing the `Tool` interface.
- Formatting logic is encapsulated with each tool.
- Easier testing, maintenance, and UI consistency.
- Supports future UIs by adding new `ui` values.

## Task Breakdown

1. **Analyze current Tool Formatter functions in both chat_view.go (TUI) and internal/ui/formatters.go (WebUI) and identify shared dependencies** (e.g., JSON parsing, diff handling).
2. **Create internal/tui/formatters package** with TUI-specific common utilities (formatDiff, formatGenericResult, etc.).
3. **Create internal/ui/formatters package** with WebUI-specific common utilities (HTML diff formatting, etc.), or extend the existing one if needed.
4. **Add FormatResult(ui string, result, diff, arguments string) string method to Tool interface** (where `ui` is "tui" or "webui" to determine output format).
5. **Add DisplayName(ui string, arguments string) (string, string) method to Tool interface** (similarly UI-aware).
6. **Implement FormatResult and DisplayName methods in all tool implementations**, returning text for TUI and HTML for WebUI.
7. **Update chat_view.go** to call `tool.FormatResult("tui", ...)` and `tool.DisplayName("tui", ...)` instead of switches.
8. **Update internal/ui/formatters.go** to call `tool.FormatResult("webui", ...)` and `tool.DisplayName("webui", ...)` instead of switches.
9. **Remove old formatter functions from both chat_view.go and internal/ui/formatters.go after migration**.
10. **Test both TUI and WebUI** to ensure formatting works correctly and consistently.

## Notes
- The `ui` parameter allows tools to return appropriate output (text for TUI, HTML for WebUI) without separate methods.
- Common parsing logic (e.g., unmarshaling JSON) remains in tools; UI-specific rendering is handled per tool.
- If preferred, separate methods (e.g., `FormatResultTUI` and `FormatResultWebUI`) could be used instead, but this unified approach reduces interface complexity.