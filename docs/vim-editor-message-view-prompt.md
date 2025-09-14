# Vim-Style Read-Only Editor for Message View

## Overview
Replace the current message view (simple viewport) with a full Vim-like editor using vimtea library, configured as read-only. This maintains all navigation capabilities while disabling data modification features.

## Requirements
- Use vimtea (github.com/kujtimiihoxha/vimtea) for Vim-like functionality
- Disable insert mode and all data-modifying commands
- Allow visual mode for selection and clipboard copying
- Preserve real-time tool event display
- Maintain existing message formatting and styling

## Current State
Messages displayed in simple viewport in `internal/tui/chat_view.go` with basic scrolling. Real-time tool events are handled via event system and displayed during processing.

## Implementation Plan

### Phase 1: Dependencies and Setup
1. Add `github.com/kujtimiihoxha/vimtea` to `go.mod`
2. Update `AGENTS.md` with vimtea configuration guidance

### Phase 2: Core Integration  
1. **Modify ChatView Structure**:
   - Replace `viewport.Model` with `vimtea.Editor`
   - Add vimtea-specific configuration fields
   - Initialize editor with message content

2. **Configure Read-Only Mode**:
   - Disable insert mode transitions (i, a, A, o, O, etc.)
   - Remove edit commands (substitute, delete, change, etc.)
   - Keep navigation bindings (hjkl, w/W, b/B, gg/G, etc.)
   - Keep visual mode bindings (v, V, Ctrl+v)
   - Configure clipboard integration for yank operations

3. **Content Population**:
   - Convert message content to string buffer for vimtea
   - Preserve existing formatting (user/assistant/system styles)  
   - Handle real-time updates during tool execution
   - Maintain message boundaries and syntax highlighting

4. **Event Integration**:
   - Update event handling to work with vimtea editor
   - Ensure real-time tool events are appended to editor content
   - Preserve existing event formatting functions

### Phase 3: Testing and Polish
1. **Unit Tests**: Test vim navigation within message view
2. **Integration Tests**: Verify real-time tool events display correctly
3. **Performance**: Ensure large chat histories don't impact responsiveness
4. **Key Bindings**: Verify all unwanted commands are properly disabled
5. **Line Numbers**: Test Ctrl+L toggle and :set number/:set nonumber commands

### Technical Considerations
- **Performance**: vimtea may need optimization for very large message buffers
- **Keyboard Conflicts**: Some vim bindings might conflict with existing TUI shortcuts (need resolution)
- **Cross-Platform**: Ensure clipboard support works on target platforms  
- **Backwards Compatibility**: Provide fallback if vimtea fails to load

### Files to Modify
- `go.mod` - Add vimtea dependency
- `AGENTS.md` - Document vimtea usage
- `internal/tui/chat_view.go` - Core replacement of viewport with vimtea
- `internal/tui/messages.go` - Update message types if needed

### Success Criteria
- Full Vim navigation (h/j/k/l, word movement, line jumps, search, etc.)
- Visual mode selection and clipboard copy working
- Line numbers can be toggled with Ctrl+L, :zn command, or :set commands
- No ability to enter insert mode or modify content
- Real-time tool events display properly during processing
- Existing message history loads and displays correctly
- Performance acceptable with large chat histories