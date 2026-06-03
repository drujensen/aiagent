---
name: security-scan
description: One-shot security scan for the aiagent project. Checks for command injection in tool execution, path traversal in file tools, API key leakage, and MongoDB injection patterns. No iteration loop — immediate findings report.
user-invocable: true
argument-hint: "scan internal/impl/tools/ for security issues"
context: fork
---

## Security Scan

Security scan for: <skill-args>

### Instructions

Use the Agent tool with `subagent_type: "security"` to perform a one-shot security scan.

Pass it:
- The files or directory to scan, or a description of the feature to review
- Any specific concerns to focus on

The security agent will:
1. Run automated grep-based scans:
   - API key/secret patterns in code
   - Command injection risks (exec.Command usage)
   - Path traversal risks (file operations)
   - MongoDB filter injection patterns
   - Sensitive data in log statements
2. Review each flagged location in context
3. Propose specific remediations for any issues found

### Output

The security agent produces:
- Automated scan results (what each grep found)
- Analysis of each finding (real vulnerability vs. false positive)
- Specific remediation for each real issue
- Summary: critical / warning / clean

### Important
- This is a one-shot scan with no review loop — use `/security-review` for the full cycle
- Critical findings (injection vulnerabilities, key leakage) should be fixed before merging
- The scan covers the aiagent threat model: BashTool injection, FileReadTool/FileWriteTool traversal, API key logging, MongoDB NoSQL injection, WebSocket validation
