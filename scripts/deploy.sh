#!/bin/bash
go build -o ./scripts/aiagent ./cmd/aiagent/main.go
sudo mv ./scripts/aiagent /usr/local/bin/aiagent
