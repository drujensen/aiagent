#!/bin/bash
go build -o ./bin/aiagent ./cmd/aiagent/main.go
sudo mv ./bin/aiagent /usr/local/bin/aiagent
