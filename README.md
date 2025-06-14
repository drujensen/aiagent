# AIAgent

## Project Description
AIAgent is a Go-based project designed to dynamically configure AI agents with various tools and prompts. It follows Domain-Driven Design (DDD) principles to maintain a clean and modular architecture.

## Installation

### Using Go Install
To install the latest version of AIAgent, run:

```bash
go install github.com/drujensen/aiagent@latest
```

### Build from Source
To build the project, ensure you have Go installed and set up on your machine. Navigate to the project directory and run:

```bash
go build -o aiagent main.go
```

## Running the Application

### Using Go Install
After installation, you can run the application:

```bash
aiagent --storage=[file|mongo]
```

### Running in Console Mode
To run in console mode, execute:

```bash
go run main.go --storage=[file|mongo]
```

### Running HTTP Server
To run the HTTP server, execute:

```bash
go run main.go --storage=[file|mongo] serve
```

Access the server at `http://localhost:8080/hello`.

## Docker and Docker Compose
### Using Docker Compose
1. Copy `.env-example` to `.env` and set your environment variables.
2. Build and start the services using Docker Compose:

```bash
docker-compose up --build
```

This will start the HTTP server, Redis, and MongoDB services.

## Testing
To test the application, you can use Go's built-in testing tools. Run the following command:

```bash
go test ./...
```

## Contribution Guidelines
We welcome contributions! Please follow these steps:
1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Commit your changes with clear messages.
4. Push your branch and create a pull request.

## Contributors
- Dru Jensen (drujensen@gmail.com)

We appreciate your interest in contributing to AIAgent! Feel free to reach out with any questions or suggestions.