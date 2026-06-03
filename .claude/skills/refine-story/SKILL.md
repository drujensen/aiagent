---
name: refine-story
description: Interactive story clarification for the aiagent project. The story-refiner agent explores the codebase, asks focused question batches, and iterates until all unknowns are resolved. Outputs a refined story with acceptance criteria and a Relevant Codebase Context section.
user-invocable: true
argument-hint: "add a new tool that does X"
context: fork
---

## Story Refinement

Refine the following story: <skill-args>

### Instructions

You are running an interactive Story Refinement session using the `story-refiner` agent.

**Step 1: Codebase Exploration**
Use the Agent tool with `subagent_type: "story-refiner"` and pass it the story description. Ask it to:
1. Explore the relevant parts of the aiagent codebase
2. Generate a focused batch of 3–6 clarifying questions — not open-ended brainstorming, specific questions with clear answers

**Step 2: Present Questions**
Present the questions to the user and collect answers.

**Step 3: Iterate**
Pass the answers back to the story-refiner agent. Ask it to:
- Either generate a follow-up batch of questions (if unknowns remain)
- Or produce the final Refined Story and Relevant Codebase Context output

**Step 4: Output**
When the story-refiner declares the story complete, present the full output to the user:
- Refined Story (Goal, Acceptance Criteria, Scope Boundaries, Known Constraints, Resolved Unknowns, Definition of Done)
- Relevant Codebase Context (table of files/services/patterns the architect and developer need to read)

### Important
- The story-refiner does NOT write code — it only asks questions and synthesizes answers
- This skill is interactive — loop with the user until the story is complete
- The output of this skill is the input to `/design-review`
