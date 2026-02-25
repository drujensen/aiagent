# Refined Plan: Add Skills Capability to AIAgent

## Updated Requirements Based on Clarifications
- **Skill Discovery**: Scan directories in priority order: `.aiagent/` (highest), then `.github/`, `.claude/`, `.codex/`, `.copilot/`, `.opencode/` (order doesn't matter). Check project root first, then home (`~/`). If duplicate skill names, prefer project version over home version.
- **Referenced Files**: Load scripts/, references/, assets/ on demand when skill content references them.
- **UI**: Sort skills alphabetically by name. Use BubbleTea List with built-in search (`/`), escape to clear filter, `q` to quit. Follow patterns from AgentView, ModelView, HistoryView.
- **Refresh**: No refresh mechanism; requires app restart.

## Architecture Design (Refined)
- **Skill Entity**: Add `internal/domain/entities/skill.go` with fields for metadata and lazy-loaded content.
- **SkillRepository**: File-based interface for discovery; implement in `internal/impl/repositories/skill.go`.
- **SkillService**: Handle business logic, validation, merging, sorting.
- **TUI Integration**: New `SkillView` following existing patterns (asynchronous loading, list.Item interface, key bindings).
- **Chat Integration**: Extend ChatService to append full skill content as user message on selection.

## Detailed Implementation Plan

### Phase 1: Domain and Infrastructure Setup
1. **Create Skill Entity** (`internal/domain/entities/skill.go`)
   - Fields: Name, Description, Path, Content (string, lazy-loaded), License, Compatibility, Metadata (map), AllowedTools ([]string)
   - Implement `list.Item` interface: `Title()` returns name, `Description()` returns description, `FilterValue()` returns name + description
   - Validation: Ensure name matches spec (lowercase, hyphens, etc.)

2. **Add SkillRepository Interface** (`internal/domain/interfaces/skill_repository.go`)
   - `DiscoverSkills(ctx) ([]Skill, error)`: Scan directories, parse SKILL.md, merge duplicates by priority

3. **Implement SkillRepository** (`internal/impl/repositories/skill.go`)
   - Define priority order: [".aiagent", ".github", ".claude", ".codex", ".copilot", ".opencode"]
   - For each skill dir in priority order, scan project root then ~/
   - Parse SKILL.md YAML frontmatter (add gopkg.in/yaml.v3 dependency if needed)
   - Use map[string]Skill to merge by name, preferring project over home, higher priority over lower
   - Error handling: Skip malformed files, log warnings for inaccessible dirs
   - Sort final list by name

4. **Create SkillService** (`internal/domain/services/skill_service.go`)
   - `ListSkills(ctx) ([]Skill, error)`: Delegate to repository
   - `GetSkillContent(ctx, skillName) (string, error)`: Load full SKILL.md + referenced files on demand
   - Validation and business logic

### Phase 2: TUI Integration
5. **Create SkillView** (`internal/tui/skill_view.go`)
   - Struct: skillService, list.Model, width, height, err
   - Follow AgentView pattern: async fetch in Init(), Update() handles WindowSizeMsg, KeyMsg (esc/q cancel, enter select)
   - Use list.NewDefaultDelegate() with selected styles
   - SetFilteringEnabled(true), SetShowStatusBar(false), SetShowPagination(true)
   - View(): Outer/inner borders, instructions ("Use arrows/j/k, / to search, Esc to clear, q to quit, Enter to execute")
   - Custom msgs: skillsFetchedMsg, skillSelectedMsg (returns skill name)

6. **Extend Main TUI** (`internal/tui/tui.go`)
   - Add skillView field and skillService dependency
   - Handle Ctrl+S key binding: switch to skill view mode
   - Handle skillSelectedMsg: call chatService to execute skill, return to chat view
   - Add skill view state management (similar to agent/model switching)

7. **Update TUI Constructor**
   - Add SkillService parameter to NewTUI()

### Phase 3: Chat Integration and Execution
8. **Extend ChatService** (`internal/domain/services/chat_service.go`)
   - Add `ExecuteSkill(ctx, skillName) error`: Get skill content from SkillService, create user message, send to current chat

9. **Wire Dependencies** (`main.go`)
   - Initialize SkillRepository and SkillService
   - Pass SkillService to TUI

### Phase 4: Testing and Validation
10. **Unit Tests**
    - Skill entity validation and list.Item methods
    - SkillRepository discovery, merging, sorting logic
    - SkillService methods
    - SkillView Update/View logic

11. **Integration Tests**
    - Full skill discovery from mock filesystem
    - TUI skill selection flow
    - Chat message creation from skill execution

12. **Edge Cases**
    - No skills found (show "No skills available")
    - Malformed SKILL.md (skip with warning)
    - Permission issues on directories (skip with warning)
    - Duplicate skills (merge by priority)
    - Skills with missing required fields (skip)

## Technical Notes
- **YAML Parsing**: Add `gopkg.in/yaml.v3` to dependencies for frontmatter parsing
- **File Loading**: For full content, read SKILL.md + any relative file references when needed
- **Performance**: Metadata-only loading for list; full content loaded only on execution
- **Error Handling**: Log errors but don't crash; show user-friendly messages in TUI
- **Styling**: Match existing views (thick outer border blue, normal inner cyan, selected cyan bold)
- **Key Bindings**: Consistent with other views (q/esc cancel, enter select, / search)

This refined plan incorporates your clarifications and follows the established TUI patterns. The implementation will be robust, user-friendly, and consistent with the existing codebase.