# Bootstrap: Fill Project Development Guidelines

## Purpose

Welcome to Trellis! This is your first task.

AI agents use `.trellis/spec/` to understand YOUR project's coding conventions.
**Empty templates = AI writes generic code that doesn't match your project style.**

Filling these guidelines is a one-time setup that pays off for every future AI session.

---

## Your Task

Fill in the guideline files based on your **existing codebase**.

### Backend Guidelines

| File | What to Document |
|------|------------------|
| `.trellis/spec/backend/directory-structure.md` | Where different file types go (routes, services, utils) |
| `.trellis/spec/backend/database-guidelines.md` | ORM, migrations, query patterns, naming conventions |
| `.trellis/spec/backend/error-handling.md` | How errors are caught, logged, and returned |
| `.trellis/spec/backend/logging-guidelines.md` | Log levels, format, what to log |
| `.trellis/spec/backend/quality-guidelines.md` | Code review standards, testing requirements |

### Thinking Guides (Optional)

The `.trellis/spec/guides/` directory contains thinking guides that are already
filled with general best practices. You can customize them for your project if needed.

---

## How to Fill Guidelines

### Principle: Document Reality, Not Ideals

Write what your codebase **actually does**, not what you wish it did.
AI needs to match existing patterns, not introduce new ones.

### Steps

1. **Look at existing code** - Find 2-3 examples of each pattern
2. **Document the pattern** - Describe what you see
3. **Include file paths** - Reference real files as examples
4. **List anti-patterns** - What does your team avoid?

---

## Tips for Using AI

Ask AI to help analyze your codebase:

- "Look at my codebase and document the patterns you see"
- "Analyze my code structure and summarize the conventions"
- "Find error handling patterns and document them"

The AI will read your code and help you document it.

---

## Completion Checklist

- [ ] Guidelines filled for your project type
- [ ] At least 2-3 real code examples in each guideline
- [ ] Anti-patterns documented

When done:

```bash
./.trellis/scripts/task.sh finish
./.trellis/scripts/task.sh archive 00-bootstrap-guidelines
```

---

## Why This Matters

After completing this task:

1. AI will write code that matches your project style
2. Relevant `/trellis:before-*-dev` commands will inject real context
3. `/trellis:check-*` commands will validate against your actual standards
4. Future developers (human or AI) will onboard faster
