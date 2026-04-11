---
description: Run the full QA workflow: fmt, vet, tidy, build, test. Fix any failures before reporting done.
---

Run the full QA workflow for the aiagent project in sequence. Stop on any failure and fix it before continuing.

```bash
go fmt ./...
go vet ./...
go mod tidy
go build .
go test ./...
```

Report:
- Which steps passed
- Any failures with the error output
- What was fixed (if anything needed fixing)

If all steps pass, confirm the project is clean.
