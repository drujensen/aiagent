# AIAgent

## Project Description
AIAgent is a Go-based project designed to dynamically configure AI agents with various tools and prompts. It follows Domain-Driven Design (DDD) principles to maintain a clean and modular architecture.

## Build Instructions
To build the project, ensure you have Go installed and set up on your machine. Navigate to the project directory and run:

```bash
cd workspace/go/aiagent
```

## Running the Application
### HTTP Server
To run the HTTP server, execute:

```bash
go run ./cmd/http
```

Access the server at `http://localhost:8080/hello`.

### Console Application
To run the console application, execute:

```bash
go run ./cmd/console
```

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
