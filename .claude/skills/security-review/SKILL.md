---
name: security-review
description: Security review loop for the aiagent project. The security agent analyzes code for vulnerabilities (command injection, path traversal, API key leakage, MongoDB injection), then the security-reviewer scores the posture against the 6 pillars. Loops until score >= 8/10 or 3 iterations maximum.
user-invocable: true
argument-hint: "review the new file tool changes for security issues"
context: fork
---

## Security Review

Security review for: <skill-args>

### Instructions

You are orchestrating a Security Review cycle for the aiagent project.

**Phase 1: Security Analysis**
1. Use the Agent tool with `subagent_type: "security"` to analyze the code
2. Pass it the feature description and list of changed files (if known)
3. The security agent will:
   - Run grep-based scans for command injection, path traversal, API key leakage, MongoDB injection patterns
   - Review each changed file against the project's threat model
   - Propose specific remediations for any issues found

**Phase 2: Review**
4. Use the Agent tool with `subagent_type: "security-reviewer"` to review the security analysis
5. Pass the security agent's full output to the reviewer
6. The reviewer will independently run the automated security scans and score against the 6 pillars

**Phase 3: Iterate**
7. If overall score **>= 8/10** (APPROVED): Report the security posture and scores
8. If overall score **< 8/10** (REVISE):
   - Extract Critical Issues from the reviewer
   - Send issues back to the security agent for remediation proposals
   - Re-review with the security-reviewer
   - Repeat until APPROVED or 3 iterations maximum

**Phase 4: Report**
9. Present:
   - Security findings summary
   - Remediations applied or recommended
   - Final pillar scores table
   - Any remaining warnings

### Important
- Maximum 3 review iterations
- The security agent PROPOSES FIXES; the reviewer only READS and SCORES
- Critical issues (injection vulnerabilities, key leakage) block shipping
- Warnings are non-blocking but should be tracked
