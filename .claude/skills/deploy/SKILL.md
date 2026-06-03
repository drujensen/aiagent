---
name: deploy
description: Deployment readiness check for the aiagent project. Verifies the release pipeline, Docker build, and all 7 binary targets are ready before tagging a release. Uses the devops agent.
user-invocable: true
argument-hint: "v1.5.0"
context: fork
---

## Deployment Readiness Check

Prepare deployment for: <skill-args>

### Instructions

Use the Agent tool with `subagent_type: "devops"` to verify deployment readiness.

Pass it the version to release (e.g., `v1.5.0`) and ask it to:

1. **Verify the codebase is ready:**
   ```bash
   go fmt ./...
   go vet ./...
   go mod tidy
   go build .
   go test ./... -race
   ```

2. **Verify cross-compilation works:**
   ```bash
   GOOS=linux GOARCH=amd64 go build -o /tmp/aiagent-linux-x86_64 .
   GOOS=linux GOARCH=arm64 go build -o /tmp/aiagent-linux-arm64 .
   GOOS=darwin GOARCH=arm64 go build -o /tmp/aiagent-macos-arm64 .
   GOOS=windows GOARCH=amd64 go build -o /tmp/aiagent-windows-x86_64.exe .
   ```

3. **Verify Docker build:**
   ```bash
   docker-compose build
   ```

4. **Check go version matches pipeline:**
   - Compare `go` directive in `go.mod` with `go-version` in `.github/workflows/release.yml`

5. **Check all 7 targets in pipeline:**
   - linux/amd64, linux/arm64, linux/riscv64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64

6. **Report the release checklist:**

```markdown
## Release Readiness: [version]

- [ ] All tests pass (`go test ./... -race`)
- [ ] Code is formatted (`go fmt ./...`)
- [ ] No vet warnings (`go vet ./...`)
- [ ] Dependencies are tidy (`go mod tidy`)
- [ ] Cross-compilation verified (at least 4 targets)
- [ ] Docker build succeeds
- [ ] Go version consistent between go.mod and pipeline
- [ ] All 7 pipeline targets present

### Next Steps
1. `git tag [version]`
2. `git push origin [version]`
3. GitHub Actions will build and upload release binaries automatically
```

### Important
- Do NOT create the git tag or push — that is the user's action
- The devops agent only verifies readiness
- If any check fails, report what needs to be fixed before releasing
