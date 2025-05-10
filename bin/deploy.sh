#!/bin/bash
go build -o ./bin/aiagent ./cmd/console/main.go
sudo mv ./bin/aiagent /usr/local/bin/aiagent
