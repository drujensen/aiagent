# Agent-Model Split Migration Guide

## Overview

**AIAgent v2.0** introduces a major architectural improvement: **Agent-Model separation**. This change provides greater flexibility but requires manual migration of existing configurations.

## What Changed

### Before (v1.x)
- Agents contained both behavior (prompts, tools) AND inference settings (model, API keys, parameters)
- Each agent was tied to a specific model/provider
- Switching models required creating new agents

### After (v2.0)
- **Agents** define behavior: prompts, tools, personality
- **Models** define inference: provider, model name, parameters, API keys
- Agents and models are independent and reusable
- Switch models or agents mid-conversation with full history preservation

## Migration Required

**⚠️ IMPORTANT**: There is **no automatic migration**. You must manually recreate your agents and models.

## Step-by-Step Migration

### Step 1: Backup Your Data
```bash
# If using file storage, backup your data directory
cp -r ~/.aiagent/data ~/.aiagent/data.backup
```

### Step 2: Understand Your Current Setup
Review your existing agents and note:
- **System prompts** and **tools** → Will become part of new Agents
- **Model settings** and **API keys** → Will become separate Models

### Step 3: Create Models First

#### Via Web UI (http://localhost:8080)
1. Navigate to **Models** section
2. Click **"Create New Model"**
3. Fill in:
   - **Name**: Descriptive name (e.g., "GPT-4 Fast", "Claude Creative")
   - **Provider**: Select from dropdown (OpenAI, Anthropic, etc.)
   - **Model Name**: Specific model (gpt-4, claude-3-sonnet, etc.)
   - **API Key**: Your provider API key
   - **Parameters**: Temperature, max tokens, context window

#### Via TUI
1. Start TUI: `aiagent`
2. Navigate to Models section
3. Create new models with your provider settings

#### Recommended Default Models
Create these essential models:
- **GPT-4 Balanced**: OpenAI, gpt-4, temp=0.7
- **Claude Fast**: Anthropic, claude-3-haiku, temp=0.3
- **Gemini Cheap**: Google, gemini-2.0-flash, temp=0.5

### Step 4: Create Agents

#### Via Web UI
1. Navigate to **Agents** section
2. Click **"Create New Agent"**
3. Fill in:
   - **Name**: Agent name (e.g., "Code Assistant", "Research Expert")
   - **System Prompt**: Your previous agent's prompt
   - **Tools**: Select tools (WebSearch, FileRead, Process, etc.)

#### Via TUI
1. Use Ctrl+A in chat to access agent management
2. Create agents with prompts and tool configurations

#### Recommended Default Agents
Create these role-based agents:

**Code Assistant Agent:**
```
System Prompt: You are an expert software developer. Help with coding tasks, debugging, and technical implementation. Use tools to examine code, run tests, and execute commands.

Tools: FileRead, FileWrite, FileSearch, Directory, Process
```

**Research Agent:**
```
System Prompt: You are a research specialist. Gather information from web sources, analyze findings, and provide comprehensive answers. Focus on accuracy and evidence-based responses.

Tools: WebSearch, FileRead, Project
```

**Planning Agent:**
```
System Prompt: You are a project planner. Break down complex tasks into manageable steps, create detailed plans, and organize work systematically.

Tools: Todo, Project, FileRead
```

### Step 5: Create Chats with Agent+Model Combinations

#### Via Web UI
1. Go to **Chats** section
2. Click **"New Chat"**
3. Select **Agent** from dropdown
4. Select **Model** from dropdown
5. Enter chat name
6. Start conversing

#### Via TUI
1. Start TUI: `aiagent`
2. Create new chat
3. Select agent + model combination
4. Begin conversation

### Step 6: Test and Refine

#### Switching During Conversations
- **Model Switching**: Use Ctrl+G (TUI) or dropdown (Web) to change models
- **Agent Switching**: Use Ctrl+A (TUI) or dropdown (Web) to change agents
- **History Preservation**: Verify chat history remains intact after switches

#### Test Different Combinations
Try these combinations:
- Code Assistant + GPT-4 (complex reasoning)
- Code Assistant + Claude Haiku (fast coding)
- Research Agent + GPT-4 (detailed analysis)
- Planning Agent + Any model (planning doesn't need advanced models)

## Troubleshooting

### Common Issues

**"Model not found" errors:**
- Ensure models are created with correct provider/model names
- Check API keys are valid for your providers
- Verify provider is supported (see Models section)

**Tools not working:**
- Ensure agent has required tools selected
- Check tool configurations in agent settings
- Verify environment variables for tool dependencies

**Chat history lost:**
- This shouldn't happen with proper switching
- Check you're using Update Chat, not creating new chats
- Verify storage configuration is correct

### Getting Help

- Check the [Testing Checklist](TESTING_CHECKLIST.md) for verification steps
- Review [AIAGENT.md](AIAGENT.md) for detailed documentation
- File issues at: https://github.com/drujensen/aiagent/issues

## Benefits After Migration

### Flexibility
- **17 model combinations** from 3 agents + 6 models (vs 3 fixed combinations before)
- Switch models for cost optimization (fast models for simple tasks, advanced for complex)
- Test same prompts across different providers

### Cost Optimization
- Use cheaper models for routine tasks
- Switch to expensive models only when needed
- Pay only for the intelligence level required

### Workflow Improvements
- **Mid-conversation switching**: Change models without restarting chats
- **Role-based agents**: Consistent behavior across different models
- **Experimentation**: Compare model performance on same tasks

## Rollback (If Needed)

If migration fails:
```bash
# Restore backup
cp -r ~/.aiagent/data.backup ~/.aiagent/data

# Downgrade to previous version
go install github.com/drujensen/aiagent@v1.x.x
```

## Questions?

This migration provides significant benefits but requires manual setup. The new architecture is much more powerful and flexible for AI agent workflows.

For questions or issues, please create an issue on GitHub or check the documentation.