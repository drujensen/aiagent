# AIAgent

[![Go](https://img.shields.io/badge/Go-1.23-blue?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

AIAgent is an open-source framework for building and interacting with AI agents. It provides a terminal user interface (TUI) and a web server, supporting multiple AI providers and a suite of powerful tools. Built with Domain-Driven Design in Go, it's modular, extensible, and easy to use.

-- NOTE: This project is in active development. Features and APIs may change frequently.  It was fully developed via vibe coding with Grok.  If you like it, I rock, if you don't, blame Grok. --

## Features

- **Multi-Mode Interfaces**:
  - Interactive Terminal User Interface (TUI) using Bubble Tea for console-based interactions.
  - Web server mode with a modern UI for browser access.
- **AI Provider Integrations**: Seamless support for OpenAI, Anthropic, Google, DeepSeek, Ollama, Groq, Mistral, Together, and xAI, with easy extensibility for more.
- **Powerful Tools**: Built-in tools including Bash execution, web search, file operations (read/write/search), directory management, image generation, vision capabilities, and more.
- **Storage Options**: Flexible persistence with file-based or MongoDB storage.
- **Chat Management**: Create, manage, and interact with multiple chat sessions, complete with message history and usage tracking.
- **Agent Configuration**: Easily define agents with custom prompts, temperature settings, tools, and context windows.
- **Docker Support**: Run the entire application with Docker Compose for easy deployment.
- **Testing and Best Practices**: Comprehensive unit tests, error handling, and code style guidelines following Go best practices.

## Architecture: Agent-Model Separation

AIAgent implements a clean separation between **Agents** and **Models** for maximum flexibility:

### Agents (Behavior)
Agents define **what** your AI assistant does:
- **System Prompt**: The personality and behavior instructions
- **Tools**: Available capabilities (web search, file operations, bash execution, etc.)
- **Name**: Human-readable identifier

Agents are **reusable** across different AI models and can be switched mid-conversation.

### Models (Inference)
Models define **how** your AI assistant thinks:
- **Provider**: OpenAI, Anthropic, Google, xAI, etc.
- **Model Name**: GPT-4, Claude-3, Gemini, Grok, etc.
- **Parameters**: Temperature, max tokens, context window
- **API Keys**: Authentication for the provider

Models can be **switched seamlessly** during conversations, with full chat history preservation.

### Key Benefits
- **Flexibility**: Mix any agent with any model
- **Cost Optimization**: Switch to cheaper/faster models for different tasks
- **Experimentation**: Test the same prompt across different models
- **Seamless Switching**: Change models mid-conversation without losing context

### Example Usage
```bash
# Create an agent with coding expertise
Agent: "Code Assistant" with tools for file operations and bash

# Use with different models based on needs:
# - GPT-4 for complex reasoning
# - Claude-3 for creative coding
# - Fast models for simple tasks
```

## Installation

### Using Go Install

To install the latest version:

```bash
go install github.com/drujensen/aiagent@latest
```

### Build from Source

Clone the repository and build:

```bash
git clone https://github.com/drujensen/aiagent.git
cd aiagent
go build -o aiagent main.go
```

To generate Swagger API documentation:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init --dir ./internal/api --output ./internal/api
```

This generates `docs.go`, `swagger.json`, and `swagger.yaml` in the `internal/api` directory, which are used to serve the Swagger UI at `http://localhost:8080/swagger/index.html` when running in web server mode.

### Using Docker

Copy `.env.example` to `.env` and configure as needed. Then:

```bash
docker-compose up --build
```

## Usage

### Terminal User Interface (TUI)

The TUI is the default mode for terminal use:

```bash
aiagent [--storage=file|mongo]
```

Navigate agents, chats, and tools interactively.

### Web Server

Run the web server:

```bash
aiagent serve [--storage=file|mongo]
```

Access at `http://localhost:8080` for a browser-based experience, including the Swagger UI at `http://localhost:8080/swagger/index.html`.

### Examples

- **Create an Agent**: Define agent behavior with prompts and tools (no model dependency)
- **Create a Model**: Configure AI models with providers, parameters, and API keys
- **Start a Chat**: Combine any agent + any model for a conversation
- **Switch Mid-Chat**: Change models or agents seamlessly while preserving history
- **Use Tools**: Agents can invoke tools like web search, file operations, and bash execution

#### Quick Start Workflow
1. **Setup**: Create agents for different roles (Coder, Researcher, Planner)
2. **Configure**: Add models from your preferred providers (OpenAI, Anthropic, etc.)
3. **Chat**: Pick agent + model combinations for different tasks
4. **Switch**: Change models during conversation for cost/speed optimization

For detailed examples, check the [documentation](AIAGENT.md).

## Configuration

- **Storage**: Use `--storage=file` for local JSON storage or `--storage=mongo` for MongoDB (configure via `.env`).
- **Environment Variables**: Set API keys and other configs in `.env` (see `.env.example`).

## Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Commit your changes with clear messages.
4. Push your branch and create a pull request.

For more details, see [CONTRIBUTING.md](CONTRIBUTING.md) (if available) or the project guidelines in [AIAGENT.md](AIAGENT.md).

## Contributors

- Dru Jensen (drujensen@gmail.com)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
