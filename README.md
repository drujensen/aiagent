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

Access at `http://localhost:8080` for a browser-based experience.

### Examples

- **Create an Agent**: Use the TUI or web UI to configure an agent with a specific AI model and tools.
- **Start a Chat**: Initiate a new chat session and interact with your AI agent.
- **Use Tools**: In a chat, invoke tools like web search or file read to enhance interactions.

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
