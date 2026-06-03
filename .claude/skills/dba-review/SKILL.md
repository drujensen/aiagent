---
name: dba-review
description: Database schema + review loop for the aiagent project. The dba agent designs schema changes ensuring JSON/MongoDB storage duality, then the dba-reviewer scores the changes against the 6 pillars. Loops until score >= 8/10 or 3 iterations maximum.
user-invocable: true
argument-hint: "add index to chat messages for faster lookup"
context: fork
---

## Database Review

Database review for: <skill-args>

### Instructions

You are orchestrating a Database Review cycle for the aiagent project.

**Phase 1: Database Work**
1. Use the Agent tool with `subagent_type: "dba"` to perform the database work
2. Pass it the story description and design document (if available)
3. The dba agent will:
   - Review the entity structs in `internal/domain/entities/`
   - Update/verify both JSON repository (`internal/impl/repositories/json/`) and MongoDB repository (`internal/impl/repositories/mongo/`)
   - Ensure struct tags are consistent: `json:"fieldName"` and `bson:"fieldName"`
   - Handle backward compatibility for existing data

**Phase 2: Review**
4. Use the Agent tool with `subagent_type: "dba-reviewer"` to review the changes
5. Pass the dba agent's output and list of changed files
6. The reviewer will verify storage duality and score against the 6 pillars

**Phase 3: Iterate**
7. If overall score **>= 8/10** (APPROVED): Report the schema changes and scores
8. If overall score **< 8/10** (REVISE):
   - Extract Critical Issues from the reviewer
   - Send issues back to the dba agent
   - Re-review with the dba-reviewer
   - Repeat until APPROVED or 3 iterations maximum

**Phase 4: Report**
9. Present:
   - Schema changes summary
   - Storage duality verification
   - Final pillar scores table

### Important
- Maximum 3 review iterations
- The dba MODIFIES FILES; the reviewer only READS and SCORES
- Every schema change must work for BOTH JSON and MongoDB
- Backward compatibility with existing data is non-negotiable
