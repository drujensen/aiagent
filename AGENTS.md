# Codebase Guidelines

## Build Commands
- Run TUI (default): `go run . [--storage=file|mongo]`
- Run web server: `go run . serve [--storage=file|mongo]`
- Run all tests: `go test ./...`
- Run specific test: `go test ./path/to/package -run TestFunctionName`
- Build and run with Docker: `docker-compose up --build`

## Development Commands
- **Lint**: `go fmt ./...` - Format Go code
- **Vet**: `go vet ./...` - Report suspicious constructs
- **Mod tidy**: `go mod tidy` - Clean up module dependencies
- **Build**: `go build .` - Compile the project
- **Test with coverage**: `go test ./... -cover` - Run tests with coverage
- **Test verbose**: `go test ./... -v` - Run tests with verbose output
- **Race detection**: `go test ./... -race` - Run tests with race detection
- **Benchmark**: `go test ./... -bench=.` - Run benchmarks

## Quality Assurance Workflow
For the Build agent to ensure code quality, run these commands in order:
1. `go fmt ./...` - Format code
2. `go vet ./...` - Check for suspicious code
3. `go mod tidy` - Clean dependencies
4. `go build .` - Compile and check for build errors
5. `go test ./...` - Run all tests

If any command fails, analyze the errors, fix the issues, and repeat the workflow until all commands pass successfully.

## Code Style Guidelines
- **Architecture**: Domain-Driven Design (DDD) with clear separation between domain and impl
- **Error Handling**: Use detailed error messages with `fmt.Errorf`, always check and propagate errors
- **Context**: Pass context in all repository and service method signatures
- **Naming**:
  - Use `NewXxx` for constructor functions
  - Interfaces should end with `er` (e.g., `Repository`, `Service`)
  - Variables should be camelCase
- **Formatting**: Run `go fmt ./...` before committing (equivalent to gopls auto-format on save)
- **Imports**: Group standard library, external, and internal imports
- **Testing**: Write unit tests for all service methods, use mocks for dependencies

## Project Structure
- `main.go`: Root entry point handling TUI (default) and web server (serve) modes with --storage flag for file or mongo
- `internal/`: Core code (domain, impl, tui, ui)
- `internal/domain/`: Business entities, interfaces, services
- `internal/impl/`: External systems integration (config, database, repositories for JSON/Mongo, tools)
- `internal/tui/`: Terminal User Interface components using Bubble Tea
- `internal/ui/`: Web UI components

## Keyboard Shortcuts & UI Flows

### TUI Shortcuts (Terminal Interface)

#### Global Shortcuts
- **Ctrl+C**: Quit application
- **Tab**: Navigate between sections (Agents, Models, Chats, etc.)
- **Enter**: Select/confirm action
- **Esc**: Cancel/go back

#### Chat Interface
- **Ctrl+A**: Switch agents in current chat (preserves history)
- **Ctrl+G**: Switch models in current chat (preserves history)
- **Ctrl+N**: Create new chat
- **↑/↓**: Navigate message history
- **Page Up/Down**: Scroll through messages

#### Agent Management
- **a**: Add new agent
- **e**: Edit selected agent
- **d**: Delete selected agent
- **Enter**: Select agent for chat

#### Model Management
- **a**: Add new model
- **e**: Edit selected model
- **d**: Delete selected model
- **Enter**: Select model for chat

#### Provider Management
- **r**: Refresh providers from models.dev API
- **Enter**: View provider details

### Web UI Flows (Browser Interface)

#### Navigation
- **Sidebar**: Click sections (Agents, Models, Chats, Providers)
- **Breadcrumbs**: Navigate back to parent sections
- **Search**: Filter lists in each section

#### Chat Interface
- **Agent Dropdown**: Switch agents mid-conversation
- **Model Dropdown**: Switch models mid-conversation
- **Send Button**: Send messages (Enter also works)
- **Scroll**: Auto-scroll to latest messages

#### CRUD Operations
- **Create**: "New" or "+" buttons
- **Edit**: Pencil icons or double-click
- **Delete**: Trash icons (with confirmation)
- **Save**: Green "Save" buttons
- **Cancel**: "Cancel" links

### Agent-Model Switching Workflows

#### TUI Workflow
1. Start chat with agent + model combination
2. Send messages to establish context
3. **Switch Model**: Press Ctrl+G → select new model → continue conversation
4. **Switch Agent**: Press Ctrl+A → select new agent → continue conversation
5. History preserved throughout switches

#### Web UI Workflow
1. Create chat with agent + model selection
2. Send messages to establish context
3. **Switch Model**: Use model dropdown → select new model → continue
4. **Switch Agent**: Use agent dropdown → select new agent → continue
5. Real-time updates without page refresh

### Vimtea Configuration
The message view uses vimtea (github.com/kujtimiihoxha/vimtea) for Vim-like navigation in read-only mode:
- **Navigation**: h/j/k/l, w/W, b/B, gg/G, search (/), etc.
- **Visual Mode**: v, V, Ctrl+v for selection and clipboard copying
- **Line Numbers**: Ctrl+L to toggle line numbers on/off, or use `:zn` command
- **Commands**: :set number/:set nonumber to control line numbers
- **Disabled**: Insert mode (i, a, A, o, O) and all editing commands (d, c, s, etc.)
- **Clipboard**: Yank operations (y) copy to system clipboard
- **Real-time Updates**: Tool events append to editor content during processing

**Note**: Vim shortcuts are only active in message view. Use TUI shortcuts elsewhere.

## Troubleshooting

### Agent-Model Split Issues

#### "Cannot create chat without agent and model"
**Problem**: Error when trying to start a chat
**Solution**:
1. Ensure you have at least one agent created (with name, prompt)
2. Ensure you have at least one model created (with provider, model name, API key)
3. In TUI: Use Tab to navigate to Agents/Models sections first
4. In Web UI: Create agents and models via their respective sections

#### Model switching not working
**Problem**: Ctrl+G doesn't switch models in TUI
**Solution**:
1. Ensure you're in an active chat (not the main menu)
2. Try refreshing the model list (navigate to Models tab and back)
3. Check that multiple models exist in your configuration
4. Restart the TUI if issues persist

#### Agent switching not working
**Problem**: Ctrl+A doesn't switch agents in TUI
**Solution**:
1. Ensure you're in an active chat
2. Verify multiple agents exist
3. Check agent configurations are valid (name and prompt required)

#### Chat history lost after switching
**Problem**: Messages disappear when changing agents/models
**Solution**:
1. This should not happen - history is preserved
2. Check you're using "switch" (Ctrl+A/Ctrl+G) not "new chat"
3. Verify storage configuration is working
4. Check logs for any repository errors

### API and Provider Issues

#### "Invalid API key" errors
**Problem**: Model requests fail with authentication errors
**Solution**:
1. Verify API key is correct for the provider
2. Check API key environment variables are set
3. Ensure model configuration uses correct API key reference
4. Test API key validity on provider's website

#### Provider refresh fails
**Problem**: `aiagent refresh` command fails
**Solution**:
1. Check internet connectivity
2. Verify models.dev API is accessible
3. Check for firewall/proxy issues
4. Try again later (service might be temporarily down)

#### Model not found in provider list
**Problem**: Specific model name not available
**Solution**:
1. Run `aiagent refresh` to update provider data
2. Check model name spelling against provider documentation
3. Verify the model is supported by your API plan
4. Some models may have geographic restrictions

### Performance Issues

#### Slow model switching
**Problem**: Delays when changing models
**Solution**:
1. Check network connectivity to AI providers
2. Verify API rate limits haven't been exceeded
3. Try switching to a different provider temporarily
4. Check system resources (CPU/memory)

#### High memory usage
**Problem**: Application uses excessive memory
**Solution**:
1. Limit number of concurrent chats
2. Close unused chat sessions
3. Restart application periodically
4. Check for memory leaks in logs

### Configuration Issues

#### Settings not persisting
**Problem**: Changes don't save between restarts
**Solution**:
1. Verify storage path permissions
2. Check storage configuration (--storage=file or --storage=mongo)
3. Ensure MongoDB is running (if using mongo storage)
4. Check disk space availability

#### Environment variables not working
**Problem**: API keys from .env not recognized
**Solution**:
1. Verify .env file exists in working directory
2. Check variable naming matches expectations
3. Restart application after .env changes
4. Ensure no typos in variable names

### Migration Issues

#### Old agents not compatible
**Problem**: Pre-v2.0 configurations don't work
**Solution**:
1. Follow the [Migration Guide](MIGRATION_GUIDE.md)
2. Manually recreate agents (behavior only) and models (inference only)
3. No automatic migration is available
4. Backup old data before migration

### Development Issues

#### Build failures
**Problem**: `go build` fails
**Solution**:
1. Run `go mod tidy` to clean dependencies
2. Ensure Go version 1.23+ is installed
3. Check for missing dependencies
4. Run `go vet` and `go fmt` for code issues

#### Test failures
**Problem**: `go test` fails
**Solution**:
1. Check test environment setup
2. Verify all dependencies are available
3. Run tests individually to isolate issues
4. Check for race conditions with `go test -race`

### Getting Help

If these solutions don't resolve your issue:

1. **Check Logs**: Enable debug logging for more details
2. **GitHub Issues**: Report bugs at https://github.com/drujensen/aiagent/issues
3. **Community**: Check existing issues for similar problems
4. **Documentation**: Review [README.md](README.md) and [AIAGENT.md](AIAGENT.md)

### Emergency Recovery

If application becomes unusable:
```bash
# Reset to defaults (WARNING: deletes all data)
rm -rf ~/.aiagent/data/*

# Restore from backup (if available)
cp -r ~/.aiagent/data.backup/* ~/.aiagent/data/

# Rebuild from source
go clean && go build
```
