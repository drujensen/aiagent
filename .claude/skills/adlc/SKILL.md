---
name: adlc
description: Full Agent Development Life Cycle pipeline for the aiagent project. Runs story refinement → design → implementation planning (with human approval checkpoint) → implementation → testing → security → database → devops. Each coding phase uses a doer + antagonistic reviewer loop requiring score >= 8/10.
user-invocable: true
argument-hint: "add a new tool that searches GitHub repositories"
context: fork
---

## ADLC Pipeline

Run the full development pipeline for: <skill-args>

---

## Stage 1: Pre-Coding (no code is written until the plan is approved)

### Step 1: Story Refinement (interactive)

Use the Agent tool with `subagent_type: "story-refiner"` to:
1. Explore the relevant aiagent codebase
2. Generate a focused batch of 3–6 clarifying questions
3. Present the questions to the user and collect answers
4. Repeat until the story is complete (all unknowns resolved)

**Capture and store the full story-refiner output** (both Refined Story and Relevant Codebase Context sections). This is passed verbatim to every subsequent agent.

When complete, ask: "Story looks complete — ready to design?"

---

### Step 2: Design (review loop)

Use the Agent tool with `subagent_type: "architect"` to design the solution. Pass it:
- The **full story-refiner output** (Refined Story + Relevant Codebase Context)

The architect defaults to maintaining existing DDD patterns. Deviations are explicitly justified.

Then use the Agent tool with `subagent_type: "architect-reviewer"` to review the design. Pass the architect's full output.

**Iterate up to 3 times** until score >= 8/10 (APPROVED) or escalate to user.

**Capture and store the approved design document** — passed verbatim to the planner.

---

### Step 3: Implementation Plan (review loop + human checkpoint)

Use the Agent tool with `subagent_type: "planner"` to produce an implementation plan. Pass it:
- The **full story-refiner output**
- The **approved design document**

The planner produces a Reuse Inventory and numbered implementation steps.

Then use the Agent tool with `subagent_type: "planner-reviewer"` to verify the plan.

**Iterate up to 3 times** until score >= 8/10.

**PAUSE. Present the complete plan to the user** (Reuse Inventory + all numbered steps).

**Ask explicitly: "Does this plan look correct? Type 'proceed' to begin coding, or describe any changes needed."**

**Do NOT advance to Stage 2 until the user types 'proceed' or equivalent confirmation.**

If the user requests changes, update the plan and re-present it.

**Capture and store the approved plan** — passed verbatim to the developer.

---

## Stage 2: Coding (only after human approves the plan)

For each coding phase, pass to the agent: the **approved plan** (including Reuse Inventory), the **approved design**, and the **refined story** (including acceptance criteria).

---

### Step 4: Implementation

**Doer:** Use the Agent tool with `subagent_type: "developer"`

Pass:
- Approved plan (Reuse Inventory + numbered steps)
- Approved design document
- Refined story (acceptance criteria)

The developer will:
- Establish a green test baseline first
- Execute one plan step at a time with test gates
- Run full QA when complete: `go fmt ./... && go vet ./... && go mod tidy && go build . && go test ./... -race`

**Reviewer:** Use the Agent tool with `subagent_type: "developer-reviewer"`

Iterate up to 3 times until score >= 8/10.

---

### Step 5: Testing

**Doer:** Use the Agent tool with `subagent_type: "tester"`

Pass:
- Refined story (acceptance criteria — each criterion should map to at least one test)
- List of files changed by the developer

**Reviewer:** Use the Agent tool with `subagent_type: "tester-reviewer"`

Iterate up to 3 times until score >= 8/10.

---

### Step 6: Security

**Doer:** Use the Agent tool with `subagent_type: "security"`

Pass:
- Refined story
- List of files changed

**Reviewer:** Use the Agent tool with `subagent_type: "security-reviewer"`

Iterate up to 3 times until score >= 8/10.

---

### Step 7: Database (run if schema or repository files were changed)

**Doer:** Use the Agent tool with `subagent_type: "dba"`

**Reviewer:** Use the Agent tool with `subagent_type: "dba-reviewer"`

Iterate up to 3 times until score >= 8/10.

---

### Step 8: DevOps (run if `.github/workflows/`, `Dockerfile`, or `compose.yml` were changed)

**Doer:** Use the Agent tool with `subagent_type: "devops"`

**Reviewer:** Use the Agent tool with `subagent_type: "devops-reviewer"`

Iterate up to 3 times until score >= 8/10.

---

### Step 9: Acceptance Criteria Verification

After all coding phases complete, verify each acceptance criterion from the refined story:

For each criterion:
- Identify: which file and/or test satisfies it, and how
- If any criterion has no corresponding implementation or test: flag it as an open gap

Use the Agent tool with `subagent_type: "developer"` or `subagent_type: "tester"` passing the refined story and the full list of changed files.

---

## Final ADLC Summary Report

Present a complete summary:

```markdown
## ADLC Pipeline Complete

### Story
**Goal:** [one sentence]
**Acceptance Criteria:**
- [x] [criterion] — satisfied by [file/test]
- [ ] [criterion] — GAP: not implemented

### Design Summary
[Key design decisions, patterns reused vs. new]

### Reuse Inventory
| Component | File Path | Used in Step |
|-----------|-----------|-------------|

### Implementation Steps Completed
[numbered list with status: completed / skipped]

### Phase Scores
| Phase | Score | Iterations | Status |
|-------|-------|-----------|--------|
| Implementation | X.X/10 | N | APPROVED |
| Testing | X.X/10 | N | APPROVED |
| Security | X.X/10 | N | APPROVED |
| Database | X.X/10 | N | APPROVED / SKIPPED |
| DevOps | X.X/10 | N | APPROVED / SKIPPED |

**Overall Score:** X.X/10 (average of completed phases)

### Files Changed
[list of modified/created files]

### Remaining Issues
[Any warnings not yet addressed]

### Ready for PR: YES / NO (if NO, list blockers)
```

---

## Important Rules

- **Maximum 3 iterations per phase** — if not approved after 3, present work with remaining issues flagged
- **Human approval is required before Stage 2** — "proceed", "yes", "looks good" counts; silence does not
- **Each doer agent CREATES/MODIFIES files; each reviewer only READS and SCORES**
- **The developer never skips test gates between steps**
- **All acceptance criteria must be verified before the summary report**
