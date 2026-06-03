---
name: devops-review
description: DevOps pipeline + review loop for the aiagent project. The devops agent manages CI/CD and Docker changes, then the devops-reviewer scores against the 6 pillars. Loops until score >= 8/10 or 3 iterations maximum.
user-invocable: true
argument-hint: "add arm64 Linux build target to the release pipeline"
context: fork
---

## DevOps Review

DevOps review for: <skill-args>

### Instructions

You are orchestrating a DevOps Review cycle for the aiagent project.

**Phase 1: DevOps Work**
1. Use the Agent tool with `subagent_type: "devops"` to perform the work
2. Pass it the story description
3. The devops agent will:
   - Modify `.github/workflows/release.yml` and/or `Dockerfile`/`compose.yml` as needed
   - Ensure all 7 build targets are maintained unless explicitly removing one
   - Verify no secrets are embedded
   - Check go version consistency with `go.mod`

**Phase 2: Review**
4. Use the Agent tool with `subagent_type: "devops-reviewer"` to review the changes
5. Pass the devops agent's output and list of changed files
6. The reviewer will verify all targets, scan for secrets, and score against the 6 pillars

**Phase 3: Iterate**
7. If overall score **>= 8/10** (APPROVED): Report the pipeline changes and scores
8. If overall score **< 8/10** (REVISE):
   - Extract Critical Issues from the reviewer
   - Send issues back to the devops agent
   - Re-review with the devops-reviewer
   - Repeat until APPROVED or 3 iterations maximum

**Phase 4: Report**
9. Present:
   - Pipeline changes summary
   - Build target verification (all 7 present?)
   - Final pillar scores table

### Important
- Maximum 3 review iterations
- The devops MODIFIES FILES; the reviewer only READS and SCORES
- Never remove a build target without explicit user confirmation
- No secrets in committed files — use GitHub Actions secrets
