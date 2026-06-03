---
name: design-review
description: Architecture design + review loop for the aiagent project. The architect designs a solution following DDD patterns, then the architect-reviewer scores it against the 6 pillars. Loops until score >= 8/10 or 3 iterations maximum.
user-invocable: true
argument-hint: "add a rate-limiting layer to AI provider calls"
context: fork
---

## Design Review

Design and review the following: <skill-args>

### Instructions

You are orchestrating an Architecture Design Review cycle for the aiagent project.

**Phase 1: Design**
1. Use the Agent tool with `subagent_type: "architect"` to design the solution
2. Pass it the story description (and the full story-refiner output if available)
3. The architect will:
   - Read the relevant codebase before designing
   - Default to maintaining existing DDD patterns
   - Produce a design document with file paths, interface definitions, and RESTful API standards where applicable

**Phase 2: Review**
3. Use the Agent tool with `subagent_type: "architect-reviewer"` to review the design
4. Pass the architect's full design document to the reviewer
5. The reviewer will score against the 6 pillars (Maintainability, Reliability, Scalability, Usability, Security, Quality)

**Phase 3: Iterate**
6. If overall score **>= 8/10** (APPROVED): Report the approved design and scores
7. If overall score **< 8/10** (REVISE):
   - Extract the specific Critical Issues from the reviewer
   - Send those issues back to the architect via Agent tool
   - Ask the architect to revise the design addressing those issues
   - Re-review with the architect-reviewer
   - Repeat until APPROVED or 3 iterations maximum

**Phase 4: Report**
8. Present:
   - Approved design document (full text)
   - Final pillar scores table
   - Iteration count and what changed between iterations
   - Note: this design is the input to `/adlc` implementation planning

### Important
- Maximum 3 review iterations to prevent infinite loops
- If not approved after 3 iterations, present the work with remaining issues flagged
- The architect DESIGNS; the architect-reviewer only READS and SCORES
- The approved design document should be saved — it is passed verbatim to the planner
