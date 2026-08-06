package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	db "transport-app/db/generated/sqlite"
	"transport-app/internal/repository"
	clockpkg "transport-app/internal/shared/clock"
	idpkg "transport-app/internal/shared/id"
	"transport-app/internal/shared/ports"
)

// OutboxWriter writes serialised domain events to the outbox database table.
type OutboxWriter struct {
	q     *db.Queries
	idGen ports.IDGenerator
	clock ports.Clock
}

// NewOutboxWriter constructs a new OutboxWriter.
func NewOutboxWriter(dbConn *sql.DB) *OutboxWriter {
	return &OutboxWriter{
		q:     db.New(dbConn),
		idGen: idpkg.NewUUIDGenerator(),
		clock: clockpkg.NewRealClock(),
	}
}

// SaveEvents serializes and inserts all events within the active transaction context.
func (w *OutboxWriter) SaveEvents(ctx context.Context, aggregateID string, aggregateType string, events []any) error {
	if len(events) == 0 {
		return nil
	}

	q := w.q
	if tx := repository.TxFromContext(ctx); tx != nil {
		q = w.q.WithTx(tx)
	}

	for _, ev := range events {
		payload, err := json.Marshal(ev)
		if err != nil {
			return err
		}

		eventType := getEventTypeName(ev)

		err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			ID:            w.idGen.GenerateUUID(),
			AggregateID:   aggregateID,
			AggregateType: aggregateType,
			EventType:     eventType,
			Payload:       string(payload),
			CreatedAt:     w.clock.Now(),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func getEventTypeName(ev any) string {
	t := fmt.Sprintf("%T", ev)
	if idx := strings.LastIndex(t, "."); idx != -1 {
		return t[idx+1:]
	}
	return t
}
