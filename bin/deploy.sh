#!/bin/bash
VERSION=$(git describe --tags --abbrev=0)
go build  -ldflags="-X 'main.version=$VERSION'" -o ./scripts/aiagent main.go
sudo mv ./scripts/aiagent /usr/local/bin/aiagent
