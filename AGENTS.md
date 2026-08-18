## Context Engine (CCE)

This project uses Code Context Engine for intelligent code retrieval and
cross-session memory.

### Searching the codebase

**Use `context_search` instead of reading files directly** when exploring
the codebase, answering questions about code, or understanding how things
work. `context_search` returns the most relevant code chunks with
confidence scores instead of whole files.

When to use `context_search`:
- Answering questions about the codebase ("how does X work?", "where is Y?")
- Exploring structure or architecture
- Finding related code, functions, or patterns

Other tools:
- `expand_chunk` for full source of a compressed result
- `related_context` for what calls/imports a function
- `session_recall` to recall past decisions

### Cross-session memory

Call `session_recall("topic phrase")` before answering non-trivial questions.
Call `record_decision(decision="...", reason="...")` after making choices.
Call `record_code_area(file_path="...", description="...")` after meaningful work.

### Output style

Respond in compressed style. Drop articles (a, an, the) in prose. Use
sentence fragments over full sentences. Use short synonyms (fix not resolve,
check not investigate). Pattern: [thing] [action] [reason]. [next step].
No filler, hedging, pleasantries, trailing summaries, or restating what
the user said. One sentence if one sentence is enough.

When suggesting code changes, show only the changed lines with 3 lines of
context. Never rewrite entire files. Multiple changes in one file: show each
change separately. Never echo back unchanged code the user already has.

Code blocks, file paths, commands, error messages: always written in full.
Security warnings and destructive action confirmations: use full clarity.

## Teach the RAG

The project has a RAG system for codebase search and custom knowledge.
**I must teach the RAG whenever I encounter knowledge worth remembering.**

### Hard rules — teach when:

1. **User shares domain knowledge** — policies, business rules, workflows, procedures
2. **I find business logic in code comments** — validation rules, state transitions, constraints
3. **User mentions something not in code** — pricing, SLAs, compliance requirements
4. **I read a doc file** — any `.md`, `.txt`, `.pdf` with business/domain content
5. **User says "teach the RAG"** — obvious, but explicit

### Don't teach when:

- Code is just boilerplate (CRUD scaffolding, imports, struct definitions)
- The knowledge is already obvious from code structure
- User hasn't asked for it and nothing new was shared

### How I teach:

```bash
# Find the content first (read the file)
# Then teach it:
bin/rag teach "topic name" file.md

# Or from stdin:
cat file.md | bin/rag teach "topic name"

# Verify:
bin/rag search "keyword from the content"
```

### Topic naming:

- Descriptive: `"booking cancellation policy"`, not `"doc1"`
- Combine related files: `rag teach "driver onboarding" docs/driver-hire.md docs/driver-checklist.md`
- Keep focused: one topic per teach call

### After teaching:

- Run `bin/rag search` to verify the content is retrievable
- Tell the user: `"taught <topic> — <n> chunks, verified"`

## AI Agent (operations assistant)

Multi-agent system in `internal/agent/` (chat at `/assistant`, API at
`/api/agent/chat`):

- **Orchestrator** (`orchestrator.go`) routes each request to a sub-agent:
  booking, payments, kharcha, ops, support (RAG knowledge_search).
- **Online RL loop** (`internal/agent/rl/`, DB `agent_rl.db`): every chat
  is an episode; rewards from tool outcomes, admin decisions, turn count.
  High-reward episodes become few-shot examples injected into system prompts;
  failing tools get deprioritization warnings. Requires `AGENT_RL_ENABLED`.
- **Approval gate** (`approval.go`): mutating tools (create_booking,
  assign_driver/vehicle, record_payment, approve/reject_kharcha) only submit
  a pending action when `AGENT_REQUIRE_APPROVAL=true`. Admin decides at
  `/agent-actions` (page) or `/api/agent/actions/{id}/approve|reject` (API).
  Actions execute under the admin's identity; decision feeds the reward.
- Tool args arrive double-encoded from OpenAI-style APIs — `normalizeArgs`
  in `agent.go` unwraps them before handlers run.
- The agent reads the acting user from `auth.ContextUser` (web session or
  bearer). RAG support tool requires `RAG_ENABLED`.

### Guardrails / rules of thumb

- Never let the agent execute mutating actions without approval configured —
  the gate is the safety boundary between "assistant" and "operator".
- The router LLM call happens first; `keywordRoute` is the fallback when the
  LLM is unavailable.
- RL data is local-first SQLite: wiping `agent_rl.db` only resets learning,
  never business data.
