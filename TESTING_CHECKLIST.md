# Agent-Model Split Manual Testing Checklist

## Overview
This checklist verifies the Agent-Model separation implementation across TUI and Web UI interfaces.

## Prerequisites
- [ ] Application builds successfully (`go build .`)
- [ ] All tests pass (`go test ./...`)
- [ ] Web server starts (`go run . serve`)
- [ ] TUI starts (`go run .`)

## TUI Testing (`go run .`)

### Agent Management
- [ ] **Agent Creation**: Create a new agent with name, prompt, and tools
- [ ] **Agent Selection**: Switch between agents using Ctrl+A
- [ ] **Agent Display**: Agent name and prompt display correctly in chat headers
- [ ] **Agent Validation**: Cannot create agent without required fields (name, prompt)

### Model Management
- [ ] **Model Creation**: Create models with provider, model name, temperature, max tokens
- [ ] **Model Selection**: Switch models using Ctrl+G in active chat
- [ ] **Model Display**: Model name displays in chat headers
- [ ] **Model Validation**: Cannot create model without required fields (provider, model name)

### Chat Functionality
- [ ] **Chat Creation**: Create chat with agent+model selection
- [ ] **Message Sending**: Send messages and receive responses
- [ ] **Model Switching**: Switch models mid-conversation, chat history preserved
- [ ] **Agent Switching**: Switch agents mid-conversation, chat history preserved
- [ ] **Chat Persistence**: Chats persist across application restarts

### Integration Features
- [ ] **Provider Refresh**: `aiagent refresh` command updates providers from models.dev
- [ ] **Default Models**: 8 default models created across different providers
- [ ] **Keyboard Shortcuts**: Ctrl+A (agents), Ctrl+G (models), Ctrl+C (quit)

## Web UI Testing (`go run . serve` → http://localhost:8080)

### Agent Management
- [ ] **Agent CRUD**: Create, read, update, delete agents via web interface
- [ ] **Agent Forms**: Name, system prompt, and tools fields work correctly
- [ ] **Agent Validation**: Form validation prevents invalid agent creation
- [ ] **Agent List**: Agents display properly in sidebar and dropdowns

### Model Management
- [ ] **Model CRUD**: Create, read, update, delete models via web interface
- [ ] **Model Forms**: Provider, model name, parameters (temp, tokens, context) work
- [ ] **Model Validation**: Form validation prevents invalid model creation
- [ ] **Model List**: Models display properly in sidebar and dropdowns

### Chat Functionality
- [ ] **Chat Creation**: Create chat with agent+model selection from dropdowns
- [ ] **Message Exchange**: Send messages and receive AI responses
- [ ] **Model Switching**: Change models in active chat via dropdown, history preserved
- [ ] **Agent Switching**: Change agents in active chat via dropdown, history preserved
- [ ] **Real-time Updates**: Messages appear immediately without page refresh

### Provider Management
- [ ] **Provider Refresh**: Refresh button updates providers from models.dev API
- [ ] **Provider Display**: Provider list shows current models and pricing
- [ ] **Provider Integration**: Models created from providers work in chats

### UI/UX Features
- [ ] **Navigation**: Sidebar navigation works between sections (agents, models, chats, providers)
- [ ] **Responsive Design**: Interface works on different screen sizes
- [ ] **Error Handling**: Proper error messages for failed operations
- [ ] **Loading States**: Loading indicators during API calls and model switches

## Cross-Platform Testing

### Data Consistency
- [ ] **JSON Storage**: Data persists correctly in JSON files
- [ ] **Entity Relationships**: Agent-Model-Chat relationships maintained
- [ ] **Migration**: No data loss when upgrading from pre-split version

### Performance
- [ ] **Fast Switching**: Model/agent switches happen quickly (< 500ms)
- [ ] **Memory Usage**: No memory leaks during extended use
- [ ] **Concurrent Access**: Multiple users can access web UI simultaneously

### Error Scenarios
- [ ] **Network Issues**: Graceful handling when models.dev API unavailable
- [ ] **Invalid API Keys**: Clear error messages for authentication failures
- [ ] **Missing Dependencies**: Proper fallbacks when optional features unavailable

## Integration Testing

### API Consistency
- [ ] **TUI ↔ Web Sync**: Changes in one interface reflect in the other
- [ ] **Real-time Updates**: Web UI updates when TUI makes changes (if applicable)
- [ ] **Session Management**: User sessions maintain state correctly

### End-to-End Workflows
- [ ] **Complete Workflow**: Create agent → Create model → Create chat → Send messages → Switch models → Continue conversation
- [ ] **Provider Workflow**: Refresh providers → Create model from provider → Use in chat
- [ ] **Management Workflow**: CRUD operations work end-to-end

## Regression Testing

### Existing Features
- [ ] **Backward Compatibility**: No breaking changes to existing functionality
- [ ] **API Endpoints**: Removed endpoints don't cause errors (previously removed)
- [ ] **Configuration**: Existing config files still work

### Edge Cases
- [ ] **Empty States**: Proper handling when no agents/models/chats exist
- [ ] **Large Datasets**: Performance with many agents/models/chats
- [ ] **Special Characters**: Names with special characters work correctly

## Final Verification

### Documentation
- [ ] **README Updated**: Agent-Model architecture clearly explained
- [ ] **User Guide**: Migration guide for existing users
- [ ] **API Reference**: Any remaining API endpoints documented

### Deployment Readiness
- [ ] **Production Build**: `go build -o aiagent .` works
- [ ] **Docker Support**: `docker-compose up --build` works (if applicable)
- [ ] **Environment Variables**: All required API keys properly configured

---

## Testing Results Summary

**Date:** __________
**Tester:** __________
**Environment:** __________

### Pass/Fail Counts
- **Total Tests:** __/__ (Pass/Fail)
- **TUI Tests:** __/__ (Pass/Fail)
- **Web UI Tests:** __/__ (Pass/Fail)
- **Integration Tests:** __/__ (Pass/Fail)

### Issues Found
1. **Issue:** ______________________
   - **Severity:** (Critical/High/Medium/Low)
   - **Steps to Reproduce:** ______________________
   - **Expected:** ______________________
   - **Actual:** ______________________

### Notes
______________________
______________________
______________________

### Sign-off
- [ ] **Code Review:** __________
- [ ] **QA Approval:** __________
- [ ] **Product Approval:** __________