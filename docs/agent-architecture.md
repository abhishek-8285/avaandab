# AI Agent (Operations Assistant) — Architecture & Operations

## What it is

The agent is an AI operations assistant ("Avandab") for transport ops. Users
chat with it in plain English; it answers questions and performs live actions
(fare quotes, bookings, payments, driver expense approvals, trip lookups)
by calling tools that talk directly to the service layer. It is an LLM-driven
tool-calling loop, not a static FAQ bot.

The acting user is read from the session (web) or bearer token (API); tools
execute under that user's identity.

## Multi-agent design

One **orchestrator** routes each request to a specialized **sub-agent**:

1. **Route** — a single LLM call classifies the user's last message into one
   sub-agent (e.g. "book a truck from Mumbai" → `booking`). Output is trimmed
   and JSON-ish replies like `{"agent":"booking"}` are tolerated.
2. **Delegate** — the chosen sub-agent's system prompt + tool set are loaded
   and a fresh `Agent` runs the tool-calling loop (max `AGENT_MAX_TURNS`
   turns; each turn is one LLM call, and tool results are fed back until the
   model stops calling tools).
3. **Record** — the whole turn (query, messages, tool traces, answer) becomes
   an RL episode (see below). If routing or the LLM fails, it falls back to
   `ops` or the keyword router.

Sub-agents and their tools:

| Agent | Handles | Tools |
|---|---|---|
| `booking` | fare quotes, routes, customers, bookings | search_routes, get_quote, search_customers, get_booking, create_booking |
| `payments` | invoices, balances, recording payments | get_invoice, list_unpaid_invoices, record_payment |
| `kharcha` | driver expense review/approvals | list_pending_kharcha, approve_kharcha, reject_kharcha |
| `ops` | trips, drivers, vehicles, assignments, dashboard, revenue | list_trips, get_trip, list_available_drivers, list_available_vehicles, assign_driver, assign_vehicle, get_dashboard, get_revenue |
| `support` | policy/how-to questions (only when RAG enabled) | knowledge_search (RAG) |

**keywordRoute fallback** — when the router LLM is unavailable or returns an
unrecognized name, a dependency-free keyword matcher picks the agent:
kharcha/expense/diesel/toll → `kharcha`; invoice/payment/upi/cheque →
`payments`; policy/rules/how does → `support`; book/quote/fare/customer →
`booking`; default → `ops`.

## Online RL loop

When `AGENT_RL_ENABLED`, every chat is recorded as an episode in a local
SQLite DB (`agent_rl.db` by default — wiping it only resets learning, never
business data). High-reward episodes become few-shot examples injected into
sub-agent prompts; tools that repeatedly fail get "deprioritization" policy
notes. RL store init failure disables learning but never the chat.

## Approval gate

Mutating tools (create_booking, assign_driver, assign_vehicle,
record_payment, approve/reject_kharcha) are gated when
`AGENT_REQUIRE_APPROVAL=true`: the agent submits a pending action instead of
executing. Admins decide at `/agent-actions` (web) or
`/api/agent/actions/{id}/approve|reject` (API). The action then executes
under the admin's identity, and the decision feeds the RL reward. This gate
is the safety boundary between "assistant" and "operator" — never run
mutating agents without it.

## normalizeArgs (double-encoding fix)

OpenAI-compatible APIs send tool `arguments` as a JSON **string** (`"{\"a\":1}"`)
while some providers send the raw **object** (`{"a":1}`). `normalizeArgs` in
`agent.go` tries to unmarshal args as a string first and re-wraps it as JSON
if it succeeds, so both shapes parse identically before any tool handler runs.

## Env vars

| Var | Default | Effect |
|---|---|---|
| `AGENT_ENABLED` | `false` | master switch; requires `AGENT_API_KEY` (logs warning and stays off otherwise) |
| `AGENT_API_KEY` | — | LLM provider key; agent won't start without it |
| `AGENT_BASE_URL` | `https://api.openai.com/v1` | OpenAI-compatible chat completions endpoint |
| `AGENT_MODEL` | `gpt-4o-mini` | model used for routing and all agent turns |
| `AGENT_MAX_TURNS` | `10` | max tool-calling loop iterations per request |
| `AGENT_RL_ENABLED` | `true` | enable online RL episode learning (SQLite) |
| `AGENT_RL_DB_PATH` | `agent_rl.db` | RL store location |
| `AGENT_REQUIRE_APPROVAL` | `true` | gate mutating tools behind the admin approval queue |

## Endpoints

- `POST /assistant/chat` — web chat, session auth (used by the assistant page)
- `GET /assistant` — the chat page UI (`assistant.html`)
- `POST /api/agent/chat` — programmatic chat, bearer/session API auth; request
  body `{"messages":[{"role":"user","content":"..."}]}`, response
  `{"reply":..., "operator":..., "episode_id":...}`
- `GET/POST /agent-actions` — admin approval queue (web)
- `/api/agent/actions/{id}/approve|reject` — approval API

Both chat endpoints hit the same handler; the orchestrator decides routing.
