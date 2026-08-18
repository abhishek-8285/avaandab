package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// ToolFunc executes a tool with parsed JSON arguments and returns a string result.
type ToolFunc func(ctx context.Context, args json.RawMessage) (string, error)

// RegisteredTool pairs a schema with its implementation.
type RegisteredTool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Handler     ToolFunc
}

// Tool implements the chat-completions tool schema.
func (t *RegisteredTool) Tool() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		},
	}
}

// Completer is the LLM chat client contract (implemented by Client).
type Completer interface {
	Complete(ctx context.Context, messages []Message, tools []Tool) (Message, error)
}

// Agent runs a tool-calling loop against an LLM.
type Agent struct {
	client   Completer
	tools    []*RegisteredTool
	maxTurns int
	system   string
	tracer   func(name string, ok bool, err error)
}

// NewAgent creates an agent with the given tools and system prompt.
func NewAgent(client Completer, tools []*RegisteredTool, systemPrompt string, maxTurns int) *Agent {
	if maxTurns <= 0 {
		maxTurns = 10
	}
	return &Agent{
		client:   client,
		tools:    tools,
		maxTurns: maxTurns,
		system:   systemPrompt,
	}
}

// WithTracer registers a callback invoked after every tool execution.
func (a *Agent) WithTracer(fn func(name string, ok bool, err error)) *Agent {
	a.tracer = fn
	return a
}

// Run executes a conversation turn and returns the final answer plus the
// number of LLM turns actually used (drives the RL turn-efficiency reward).
func (a *Agent) Run(ctx context.Context, history []Message) (string, int, error) {
	messages := make([]Message, 0, len(history)+1)
	if a.system != "" {
		messages = append(messages, Message{Role: "system", Content: a.system})
	}
	messages = append(messages, history...)

	tools := make([]Tool, 0, len(a.tools))
	for _, t := range a.tools {
		tools = append(tools, t.Tool())
	}

	turnCount := 0
	for turn := 0; turn < a.maxTurns; turn++ {
		turnCount++
		reply, err := a.client.Complete(ctx, messages, tools)
		if err != nil {
			return "", turnCount, fmt.Errorf("LLM call %d: %w", turn, err)
		}
		messages = append(messages, reply)

		if len(reply.ToolCalls) == 0 {
			if reply.FinishReason == "tool_calls" {
				return "", turnCount, fmt.Errorf("LLM reply %d: finish_reason 'tool_calls' with no tool calls (protocol error)", turn)
			}
			if reply.FinishReason == "length" {
				return "", turnCount, fmt.Errorf("LLM reply %d: response truncated (finish_reason 'length')", turn)
			}
			return reply.Content, turnCount, nil
		}

		for _, call := range reply.ToolCalls {
			result, err := a.executeTool(ctx, call.Function)
			if a.tracer != nil {
				a.tracer(call.Function.Name, err == nil, err)
			}
			if err != nil {
				log.Printf("agent: tool %s error: %v", call.Function.Name, err)
				result = "error: " + sanitizeToolError(err)
			}
			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Name:       call.Function.Name,
				Content:    result,
			})
			log.Printf("agent: tool %s -> %s", call.Function.Name, truncate(result, 200))
		}

		// Last permitted turn produced tool calls: give the model one final
		// chance to answer with the results instead of failing the chat.
		if turn == a.maxTurns-1 {
			final, err := a.client.Complete(ctx, messages, nil)
			if err != nil {
				return "", turnCount, fmt.Errorf("final LLM call: %w", err)
			}
			if strings.TrimSpace(final.Content) == "" {
				return "", turnCount, fmt.Errorf("agent exceeded %d turns (used %d): empty final answer", a.maxTurns, turnCount)
			}
			return final.Content, turnCount, nil
		}
	}

	return "", turnCount, fmt.Errorf("agent exceeded %d turns (used %d)", a.maxTurns, turnCount)
}

// sanitizeToolError keeps tool error detail out of the model context where it
// could be echoed to the user (schema/DB internals). Newlines and over-long
// messages are stripped; the raw error is already logged above and traced.
func sanitizeToolError(err error) string {
	if err == nil {
		return ""
	}
	s := strings.Join(strings.Fields(err.Error()), " ")
	r := []rune(s)
	if len(r) > 300 {
		return string(r[:300]) + "..."
	}
	return s
}

func (a *Agent) executeTool(ctx context.Context, fn FunctionCall) (string, error) {
	if len(fn.Arguments) == 0 || string(fn.Arguments) == "null" {
		return "", fmt.Errorf("tool %q called without arguments", fn.Name)
	}
	for _, t := range a.tools {
		if t.Name != fn.Name {
			continue
		}
		return t.Handler(ctx, normalizeArgs(fn.Arguments))
	}
	return "", fmt.Errorf("unknown tool %q", fn.Name)
}

// normalizeArgs unwraps double-encoded tool arguments. OpenAI-compatible APIs
// send "arguments" as a JSON string ("{...}") while some providers send the
// raw object ({...}); both must parse identically.
func normalizeArgs(args json.RawMessage) json.RawMessage {
	for depth := 0; depth < 3; depth++ {
		var s string
		if json.Unmarshal(args, &s) != nil {
			return args
		}
		args = json.RawMessage(s)
	}
	return args
}

const (
	userIDCtxKey   ctxKey = "agent_user_id"
	userNameCtxKey ctxKey = "agent_user_name"
)

// userIDFrom returns the acting user id from context ("" when absent).
func userIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(userIDCtxKey).(string); ok {
		return v
	}
	return ""
}

// userNameFrom returns the acting user's display name from context.
func userNameFrom(ctx context.Context) string {
	if v, ok := ctx.Value(userNameCtxKey).(string); ok {
		return v
	}
	return ""
}

// SystemPromptFor builds a system prompt bound to the operator's role.
func SystemPromptFor(operatorName string) string {
	var b strings.Builder
	b.WriteString(`You are the Avandab AI operations assistant inside a multi-vehicle transport management system. You help dispatchers, accountants and admins run daily operations.

You have tools to interact with the live system. Follow these rules:

1. Answer in plain, concise English. Use INR for money.
2. When a user asks for a quote: find the route first, then compute fare = standard fare (+18% GST if the system has gst enabled), mention estimated hours and distance.
3. To create a booking you need: customer id (or name to search), route (source+destination), pickup date/time, vehicle type, passengers, price. Ask for anything missing.
4. When asked about "today's trips", list status, trip number, route and driver.
5. Before assigning a driver or vehicle, confirm the trip id/number with the user if not given.
6. After any mutation (booking created, driver assigned, payment recorded), state exactly what was done, with IDs/numbers.
7. Never invent IDs or data. If a lookup returns nothing, say so and suggest next steps.
8. You can check pending kharcha expenses and approve/reject them when asked.
9. Keep tool calls minimal — prefer a single call that answers the question.

Your authenticated operator is: ` + operatorName)

	return b.String()
}
