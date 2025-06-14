#!/bin/bash
go build -o ./scripts/aiagent main.go
sudo mv ./scripts/aiagent /usr/local/bin/aiagent