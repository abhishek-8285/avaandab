# Agent Approval Gate

## Purpose

Mutating agent tools never execute directly. When the AI assistant wants to
change state, it submits a **pending action** to the approval queue instead.
An admin reviews and either approves or rejects it. Only then does the
mutation run. This is the safety boundary between "assistant" and
"operator": the agent proposes, the admin disposes.

## Gated tools

Six tools are approval-gated when the gate is on: `create_booking`,
`assign_driver`, `assign_vehicle`, `record_payment`, `approve_kharcha`,
`reject_kharcha`. All other agent tools (search, lookup, RAG) run directly.

## Flow

1. User chats with the assistant (`/assistant` or `/api/agent/chat`).
2. Agent picks a gated tool. Instead of running, the wrapped handler submits
   an action (tool name, raw args, human-readable summary) and replies:
   `Action act-... submitted for admin approval: <summary>. Pending decision
   in the approval queue.`
3. The action sits in the queue as `pending`.
4. Admin approves or rejects. The outcome feeds the RL reward loop
   (`agent_rl.db`).

## Where admins decide

Both surfaces are admin-only.

- **Web page** `GET /agent-actions` — session auth, admin role. Lists pending
  actions (tool, summary, raw args, requester). Approve posts to
  `/agent-actions/{id}/approve`; reject opens a modal and posts to
  `/agent-actions/{id}/reject` with a required `reason`. Non-admins get
  redirected to `/dashboard` (page) or 403 (posts).
- **API** `POST /api/agent/actions/{id}/approve` and
  `POST /api/agent/actions/{id}/reject` — bearer token (or session) auth,
  admin role. Reject requires JSON `{"reason": "..."}`; both return the
  updated action. `GET /api/agent/actions?status=...` lists actions.

Both endpoints only accept `pending` actions; anything else returns 400.

## Env var

`AGENT_REQUIRE_APPROVAL` — default **true** (gate on). Set `false` to let
the agent act directly: mutating tools run immediately, no queue, no
sign-off. Only in trusted/non-prod environments.

## Execution on approval

On approve, the original tool handler runs with the **admin's identity**:
`ToolEnv.UserID`/`UserName` are swapped to the approving admin for the
duration of the call (so kharcha approvals record the correct approver),
then restored. The decision is attributed to the admin on the action.

## Action statuses

- `pending` — submitted, awaiting decision
- `approved` — admin said yes
- `rejected` — admin said no, with reason
- `executed` — handler ran successfully
- `failed` — handler returned an error (stored on the action)

## Web CSRF note

Web POSTs (the `/agent-actions/*` forms) go through the global CSRF
middleware: cross-site state-changing requests carrying a session cookie are
rejected unless they present a matching `Origin` header (complements
SameSite=Lax). Bearer-token API calls are unaffected by the Origin check.
