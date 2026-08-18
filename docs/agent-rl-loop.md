# Agent Online RL Loop

The assistant is a multi-agent system (booking, payments, kharcha, ops,
support). The orchestrator routes each chat to a sub-agent and runs an
**online reinforcement-learning loop**: every conversation is logged as an
episode, scored with reward signals, and the outcomes shape future prompts.

This is **not model fine-tuning**. No weights are updated. Learning is
local-first and prompt-level: high-reward episodes are replayed as few-shot
examples and unreliable tools get deprioritization warnings. All state lives
in a single SQLite database, `agent_rl.db` (requires `AGENT_RL_ENABLED`).

## Flow per chat

1. Orchestrator routes the query to a sub-agent (LLM router; keyword fallback).
2. System prompt is built from the sub-agent's base prompt plus anything
   learned: matching few-shot examples and tool warnings.
3. Agent runs with a tracer; every tool call records name, success, error,
   duration.
4. Episode (query, answer, messages, traces) is saved and rewards applied.
5. Mutating tools (create_booking, assign_driver/vehicle, record_payment,
   approve/reject_kharcha) don't execute directly — they submit a pending
   action to the admin approval queue.

## `agent_rl.db` tables

- **agent_episodes** — one row per chat: agent, user, query, answer, reward
  total, status (pending → rewarded), messages/traces JSON. Indexed by
  (agent_name, reward).
- **agent_actions** — mutating actions awaiting or after admin decision:
  tool, args, summary, requester, decider, status, result/error.
- **agent_rewards** — individual reward signals per episode with value and
  note.
- **agent_tool_stats** — per (agent, tool) call and failure counts, PK on
  agent+tool.

## Reward signals

Implicit (from the episode itself):

| Signal | Value | When |
|---|---|---|
| tool_ok | +0.15 | each tool call succeeded |
| tool_error | -0.5 | each tool call failed |
| turns_efficient | +0.2 | 1–2 tool turns total |
| turns_wasteful | -0.15 | more than 4 tool turns |
| answer_empty | -0.3 | agent returned an empty answer |

From admin decisions on approval-gated actions:

| Signal | Value | When |
|---|---|---|
| action_executed | +1.0 | approved and executed successfully |
| action_failed | -1.5 | approved but execution failed |
| action_rejected | -0.8 | admin rejected |

Rewards accumulate on the episode total (each signal row is also stored in
agent_rewards).

## Preference memory (few-shot)

Episodes with reward >= 0.5 and status `rewarded` are candidates for replay.
On each request, up to 50 top episodes for the agent are scored against the
current query by **bigram overlap** (character-level n-gram similarity on
lowercased words); the top 3 with any overlap are injected into the sub-agent
system prompt as "reference examples of how similar requests were handled
well", including which tools were used. No similarity → no injection.

## Policy shaping

Per-agent tool stats are checked before each chat. Tools with >= 3 calls and
> 40% failure rate get a warning in the system prompt: "only use as last
resort", with the failure count. This steers the agent away from tools that
keep breaking.

## Admin decisions feed rewards

The approval queue is where human judgment enters the loop:

- **Approve**: action runs under the admin's identity. Success → +1.0
  action_executed; the handler failing → -1.5 action_failed.
- **Reject**: no execution → -0.8 action_rejected.

The decision is stored on the action and its reward applied to the owning
episode, so the agent learns which proposed actions an admin will accept.
Decision also feeds the RL reward model for the episode.

## Resetting learning

Delete `agent_rl.db` (plus any `-wal`/`-shm` sidecars) while the app is
stopped. This wipes episodes, actions, rewards, and tool stats — learning
starts fresh. Business data is untouched; agent_rl.db is purely local
learning state.
