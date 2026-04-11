---
description: Create a new skill for the aiagent project under .aiagent/skills/. A skill is a reusable prompt/instruction set that agents can invoke.
---

Create a new aiagent skill.

Skill name and purpose: $ARGUMENTS

## Skill File Format

Skills are discovered from `.aiagent/skills/<name>/SKILL.md`. The format requires YAML frontmatter:

```markdown
---
name: <skill-name>
description: <description up to 1024 chars>
license: MIT
compatibility: aiagent>=1.0
allowed-tools:
  - bash
  - file-read
metadata:
  author: drujensen
---

# Skill content here (the actual instructions/prompt for the agent)
```

## Name Rules
- Lowercase letters, numbers, hyphens only
- Cannot start or end with a hyphen
- No consecutive hyphens
- Max 64 characters

## Steps

1. Determine the skill name (snake-case to hyphen-case)
2. Create directory: `.aiagent/skills/<name>/`
3. Create `.aiagent/skills/<name>/SKILL.md` with proper frontmatter
4. Write clear, actionable skill content that an AI agent can follow
5. Verify the skill would pass `skill.Validate()` (name format, description length)

The skill content should be a self-contained set of instructions that an agent can execute. Be specific about what tools to use and what output to produce.
