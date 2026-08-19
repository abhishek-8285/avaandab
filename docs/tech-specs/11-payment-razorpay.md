# REAL Razorpay Payments — Implementation Spec v1

Status: ready
Depends-on: 00-migration-ownership-index.md (reserve 00057), migration 00037 (razorpay method already in DB CHECK), 00035 (idempotency_key column)
Migration owner: db/migrations/00057_*.sql (Razorpay payment columns + webhook idempotency indexes)

---

## 0. Verified ground truth (file:line)

Facts gathered by reading the repo — do NOT trust prose, trust these lines.

- **UI button is cosmetic only.** `internal/templates/payment_edit.html:51-78` opens Razorpay checkout with NO `order_id` and on success (`handler`) just stuffs the payment id into `reference` and force-sets `method = "upi"`, then submits the normal form. It **never** verifies a signature and never sets `method = "razorpay"`.
- **Select lacks razorpay option.** `internal/templates/payment_edit.html:23-29` options are cash/upi/bank_transfer/cheque only.
- **No server-side order is ever created**, so `notes.invoice_id` is never attached. This is why the webhook below is broken.
- **Webhook is real but broken in practice.** `internal/payment/application/razorpay_webhook.go:232-256` `recordPaymentEntity` returns `ErrWebhookInvoiceMissing` when `entity.Notes.InvoiceID == ""`. Since the UI never creates a server order, Razorpay never forwards `notes.invoice_id`, so every `payment.captured` webhook is dropped. Grep proof:
  - `grep -rn "CreateOrder" internal/ cmd/ ` → only `internal/payment/razorpay/client.go:41` (DEAD, never called).
  - **`CreateOrder` + `VerifyPaymentSignature` are wired only into non-server code.** Grep shows no caller in the HTTP server: `internal/payment/razorpay/client.go:41,78` are invoked solely by `cmd/test_razorpay/main.go:19,23,38` and `internal/payment/razorpay/client_test.go:23,27`. They are NOT wired into `cmd/server/main.go` or any request handler — the production payment path never creates an order or verifies a signature server-side.
- **Webhook uses its OWN HMAC verification** (`razorpay_webhook.go:148-165` `VerifySignature` over raw body with `uc.secret`). This is correct for webhooks. The client.go `VerifyPaymentSignature` (order_id|payment_id HMAC) is the *checkout redirect* verifier and is the one we must wire into the new `/verify` endpoint.
  - **Idempotency has TWO layers — one broken, one already working.** `razorpay_webhook.go:103-123` `processedIDs map[string]processedEvent` with `webhookEventTTL = 24h` (`razorpay_webhook.go:22`) is the **eventID** layer: in-memory only, wiped on restart, so a restart can re-process the same event. BUT the **razorpay_payment_id** layer is ALREADY DB-backed: `recordPaymentEntity` sets `Reference = entity.ID` (`razorpay_webhook.go:247,254`), `RecordPaymentUseCase` derives `idempotency_key = "ref:"+reference` (`payment_repository.go:170-176`), `FindByReference` looks it up (`payment_repository.go:147-156` → `findIDByIdempotencyKey`), enforced by UNIQUE `idx_payments_idempotency` (`00035`). So a duplicate `payment.captured` for the same Razorpay payment id is already deduped at the DB across restarts; the genuine restart gap is only for missing/duplicate `eventID`.
- **DB already allows method='razorpay'** via migration `00037_payments_razorpay_method.sql:12` (CHECK includes `'razorpay'`). Schema columns `tenant_id` and `idempotency_key` already exist (`00037:17-18`, `00035:2`).
- **Legacy enum missing razorpay.** `internal/domain/payment/entity.go:25-30` defines only cash/upi/bank_transfer/cheque. No `PaymentMethodRazorpay`. The aggregate copy HAS it: `internal/payment/domain/aggregate/payment_aggregate.go:17`. The legacy one is exposed via `internal/domain/aliases.go:78,160-163` and consumed by `internal/repository/sqlite/payments.go:82,189` and `internal/agent/tools.go:579`. This is a divergence we must align.
- **API handler already accepts razorpay method in Record.** `internal/payment/presentation/api/handlers/payment_handler.go:118` case includes `aggregate.PaymentMethodRazorpay`.
- **Config vars exist.** `internal/config/config.go:26-28` `RazorpayKeyID`, `RazorpayKeySecret`, `RazorpayWebhook` (webhook secret). Loaded at `:119-121` from `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET`, `RAZORPAY_WEBHOOK_SECRET`.
- **Webhook route mounted publicly.** `cmd/server/main.go:441` `r...Post("/api/v1/payments/razorpay-webhook", ...)` (rate-limited, no session). The new order/verify routes must be inside the protected group (`cmd/server/main.go:443-455`) with RBAC `payments:create`.
- **Webhook UC constructed** at `cmd/server/main.go:257` with `cfg.RazorpayWebhook` secret. Reverse UC wired at `payment_handler.go:50-52`.

### Non-goals
- We do NOT build the agent/tools Razorpay flow here (separate concern).

---

## Verification Log (Principal Engineer QA pass)

Each claim above was checked against source on 2026-08-19. Verdicts: ✅ verified / ❌ wrong.

| # | Claim (short) | Verdict | Correction / Evidence |
|---|--------------|---------|------------------------|
| 1 | UI button cosmetic, no `order_id`, forces `method=upi` | ✅ | `payment_edit.html:51-78` opens checkout w/o `order_id`; handler sets `reference`+`method=upi`, submits form. No signature verify. |
| 2 | Select lacks `razorpay` option | ✅ | `payment_edit.html:24-27` options cash/upi/bank_transfer/cheque only. |
| 3 | No server order created → `notes.invoice_id` never set | ✅ | `CreateOrder` only in `cmd/test_razorpay/main.go` + `client_test.go`; none in HTTP server. |
| 4 | Webhook broken: `recordPaymentEntity` → `ErrWebhookInvoiceMissing` on empty `notes.invoice_id` | ✅ | `razorpay_webhook.go:233-235`; root cause is claim 3. |
| 5 | `CreateOrder`/`VerifyPaymentSignature` dead | ❌ (evidence wrong) | They ARE called by `cmd/test_razorpay/main.go:23,38` + `client_test.go:23,27`. Correct: not wired into the **server**. |
| 6 | Webhook uses own HMAC `VerifySignature` over raw body | ✅ | `razorpay_webhook.go:148-165`; correct for webhooks. |
| 7 | Idempotency in-memory only; **NO DB dedup by `razorpay_payment_id`** | ❌ (2nd half wrong) | `razorpay_payment_id` IS deduped via `reference`→`idempotency_key` UNIQUE `idx_payments_idempotency` (`00035`, `payment_repository.go:147-176`). Only the `eventID` layer is restart-unsafe. |
| 8 | DB CHECK allows `razorpay` | ✅ | `00037` CHECK includes `'razorpay'`. |
| 9 | `tenant_id`/`idempotency_key` columns exist | ✅ | `00035`, `00037` recreate table with both. |
| 10 | Legacy enum missing `razorpay` | ✅ | `entity.go:25-30` lacks it; aggregate has it (`payment_aggregate.go:17`). |
| 11 | API handler accepts `razorpay` in Record | ✅ | `payment_handler.go:118` case includes `PaymentMethodRazorpay`. |
| 12 | Config vars exist | ✅ | `config.go:26-28`; loaded `:119-121` from `RAZORPAY_*`. |
| 13 | Webhook route public | ✅ | `main.go:441` outside protected group (`:443-455`). |
| 14 | Webhook UC constructed + reverse wired | ✅ | `main.go:257`; `payment_handler.go:50-52`. |

**Wrong claims: 2** — #5 (grep proof incomplete) and #7 ("no DB dedup by razorpay_payment_id" is false). Both corrected in §0 above.

### Explicit Decisions (Tradeoff + Cost)

- **Webhook HMAC verification.** Keep `razorpay_webhook.go:148-165` raw-body HMAC-SHA256 with `cfg.RazorpayWebhook` as the single verifier. *Decision:* trust the existing impl; do NOT add a second verifier. *Tradeoff:* body must be read raw (already done) — any future compression/middleware altering the body breaks it. *Cost:* near-zero; risk stays only if a proxy rewrites the body.
- **Idempotency store (DB vs in-memory).** *Decision:* rely on the existing `idempotency_key` UNIQUE index for `razorpay_payment_id` dedup (already restart-safe) and ADD `webhook_event_id` column + `idx_payments_webhook_event` (migration 00057) to make the `eventID` layer restart-safe too. Keep the in-memory map ONLY as a hot read-cache. *Tradeoff:* webhook path now does 1 extra indexed lookup per event; minor write amplification on 00057 columns. *Cost:* one new migration (00057) + converter mapping; no behavior regressions since existing `reference`/idempotency dedup is preserved. The `idx_payments_razorpay_payment` in 00057 is partially redundant with `idx_payments_idempotency` but gives a dedicated, queryable column — accepted.
- **Refunds.** *Decision:* keep `refund.processed` → `reverseUC.Execute` (`razorpay_webhook.go:284-308`); stamp the reversing row's `Reference` with the Razorpay payment/refund id so it reuses the same `idempotency_key` dedup (no double-reverse). Keep original row; add negative reversing row for partial refunds. *Tradeoff:* no separate `razorpay_refund_id` column (reuses `reference`/`idempotency_key`) — acceptable until multi-refund reporting is needed. *Cost:* none beyond existing; verify `FindByReference` blocks a replayed `refund.processed`.
- We do NOT replace the existing `Record` endpoint; we ADD order + verify + webhook persistence.
- We do NOT implement recurring/subscriptions.

---

## 1. Overview / goal

Make Razorpay a **real, server-authoritative** payment path:

1. Server creates a Razorpay Order (`notes.invoice_id` set) → returns `order_id` + publishable key to the browser.
2. Browser opens checkout with that `order_id`; on success calls a **server verify** endpoint that HMAC-verifies (`order_id|payment_id` vs `razorpay_signature`) and records the payment with `method = "razorpay"`.
3. Razorpay webhook (`payment.captured`, `order.paid`, `refund.processed`, `payment.failed`) is made **restart-safe and idempotent** via persisted `webhook_event_id` + `razorpay_payment_id` indexes, so a duplicate/late/restarted delivery never double-counts.

Explicit non-goals: manual UPI/QR collection UI (future, §8), toll auto-collect (future, §8).

---

## 2. API contract

All payment routes live under the protected group (`cmd/server/main.go:443-455`). RBAC resource `payments`.

### 2.1 POST /api/v1/payments/razorpay-order
Auth: session OR Bearer, permission `payments:create`.
Creates a server-side Razorpay Order and persists a **pending** payment row (status `pending`) so the verify step can reconcile. Returns the order id + publishable key.

Request JSON:
```json
{ "invoice_id": "inv_123", "amount": 1250.00 }
```
- `amount` optional; if omitted/absent the server reads the invoice **outstanding balance** from `h.Services.Invoices.GetBalance` (same source the UI uses at `internal/handlers/payments.go:91`).
- Reject if invoice not found (404) or balance <= 0 (400 `invoice_already_paid`).

Response 200:
```json
{
  "order_id": "order_QaBcDeFgHiJk",
  "razorpay_key_id": "rzp_live_xxxxxxxx",
  "amount_paise": 125000,
  "currency": "INR",
  "invoice_id": "inv_123"
}
```
Errors: 400 invalid body, 404 invoice not found, 402 invoice already paid, 503 razorpay not configured (`RAZORPAY_KEY_ID/SECRET` empty).

### 2.2 POST /api/v1/payments/razorpay/verify
Auth: session OR Bearer, permission `payments:create`.
Verifies the checkout signature **server-side** (single source of truth — never trust the client), then records the payment. This replaces the broken "stuff payment_id into reference + force method=upi" UI hack.

Request JSON:
```json
{
  "invoice_id": "inv_123",
  "razorpay_order_id": "order_QaBcDeFgHiJk",
  "razorpay_payment_id": "pay_AbCdEfGhIjKl",
  "razorpay_signature": "a1b2c3...hex..."
}
```

Server steps:
1. `razorpayClient.VerifyPaymentSignature(order_id, payment_id, signature)` (existing `internal/payment/razorpay/client.go:78`). If false → 401 `invalid_signature`.
2. Idempotency: if a payment with `razorpay_payment_id = <payment_id>` already exists for tenant → return existing id (200, no double-record).
3. `recordUC.Execute(...)` with `Method = aggregate.PaymentMethodRazorpay`, `Reference = &razorpay_payment_id`, and set `razorpay_order_id`, `razorpay_signature` on the row.
4. Mark invoice paid/partial via existing invoice service (same path the legacy `Create` uses after recording — `internal/handlers/payments.go:135-143`).

Response 201:
```json
{ "id": "pay_xxx", "method": "razorpay", "amount": 1250.00, "invoice_id": "inv_123" }
```
Errors: 400 missing field, 401 invalid_signature, 404 invoice not found, 409 already recorded (return existing id), 503 razorpay not configured.

### 2.3 Webhook (existing) — make restart-safe
Route `POST /api/v1/payments/razorpay-webhook` (`cmd/server/main.go:441`) stays public. Change inside `razorpay_webhook.go`:
- Persist processed `event_id` into `webhook_event_id` column and dedup by `razorpay_payment_id` (column) instead of relying only on the in-memory map. Keep the in-memory map as a hot cache, but the DB is the source of truth on restart.
- `payment.captured` / `order.paid`: read `notes.invoice_id` — now populated because orders are created server-side (§5). If still missing → 200 OK (ack) but log + emit `RazorpayPaymentFailed`/alert, do not 4xx (Razorpay retries on non-2xx).
- `refund.processed`: reverse payment (already implemented `processRefundEntity` `razorpay_webhook.go:284-308`).
- `payment.failed`: emit event (already `processFailed` `razorpay_webhook.go:310-331`).

### 2.4 PaymentDTO additions
`internal/payment/presentation/api/dto/payment_dto.go` add:
```go
RazorpayOrderID     *string `json:"razorpay_order_id,omitempty"`
RazorpayPaymentID   *string `json:"razorpay_payment_id,omitempty"`
RazorpaySignature   *string `json:"razorpay_signature,omitempty"`
WebhookEventID      *string `json:"webhook_event_id,omitempty"`
```
Populate these from the repository converter (`internal/payment/infrastructure/persistence/sql/converters/payment_converter.go`).

---

## 3. DB contract — migration 00057

Owner: `db/migrations/00057_razorpay_payment_fields.sql`. Adds Razorpay columns + restart-safe idempotency indexes. `tenant_id` and `idempotency_key` already exist (00037/00035) so we do not recreate them.

```sql
-- +goose Up
-- Razorpay traceability + restart-safe webhook idempotency.
ALTER TABLE payments ADD COLUMN razorpay_order_id   TEXT;
ALTER TABLE payments ADD COLUMN razorpay_payment_id TEXT;
ALTER TABLE payments ADD COLUMN razorpay_signature  TEXT;
ALTER TABLE payments ADD COLUMN webhook_event_id    TEXT;

-- Restart-safe webhook dedup: a given Razorpay event is processed once.
CREATE UNIQUE INDEX idx_payments_webhook_event ON payments(tenant_id, webhook_event_id)
  WHERE webhook_event_id IS NOT NULL;

-- Idempotent verify: a given Razorpay payment records once even if
-- /verify is called twice or webhook races the verify call.
CREATE UNIQUE INDEX idx_payments_razorpay_payment ON payments(tenant_id, razorpay_payment_id)
  WHERE razorpay_payment_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_payments_webhook_event;
DROP INDEX IF EXISTS idx_payments_razorpay_payment;
ALTER TABLE payments DROP COLUMN razorpay_order_id;
ALTER TABLE payments DROP COLUMN razorpay_payment_id;
ALTER TABLE payments DROP COLUMN razorpay_signature;
ALTER TABLE payments DROP COLUMN webhook_event_id;
```

Notes:
- Unique partial indexes are the cleanest dedup: `NULL` values are not indexed, so a normal cash payment (no razorpay fields) is never constrained.
- No seed rows needed. `company_config` is owned by 00042 — do NOT touch it.

### Align legacy enum
Add `PaymentMethodRazorpay` to the legacy entity so the two enums agree:

`internal/domain/payment/entity.go:25-30` add:
```go
const (
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodUPI          PaymentMethod = "upi"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCheque       PaymentMethod = "cheque"
	PaymentMethodRazorpay     PaymentMethod = "razorpay"
)
```
`internal/domain/aliases.go:163` add:
```go
PaymentMethodRazorpay      = payment.PaymentMethodRazorpay
```

---

## 4. UI

File: `internal/templates/payment_edit.html`.

1. Add the razorpay option to the select (`:23-29`):
```html
<option value="cash">Cash</option>
<option value="upi">UPI</option>
<option value="bank_transfer">Bank Transfer</option>
<option value="cheque">Cheque</option>
<option value="razorpay">Razorpay (Online)</option>
```
2. Replace the cosmetic button + JS (`:51-78`) with a real order→verify→reload flow. The button calls `/api/v1/payments/razorpay-order`, opens checkout with the returned `order_id`, and on success POSTs to `/api/v1/payments/razorpay/verify`, then reloads the invoice page.

```html
<button type="button" id="razorpay-btn" class="px-4 py-2 min-h-[44px] bg-status-success text-on-primary font-medium rounded-md hover:bg-status-success/90 ...">
    💳 Pay Online via Razorpay
</button>
```
```html
<script src="https://checkout.razorpay.com/v1/checkout.js"></script>
<script>
document.getElementById('razorpay-btn').onclick = async function (e) {
    e.preventDefault();
    const invoiceId = "{{.InvoiceID}}";
    const balance = parseFloat("{{if gt .Balance 0.0}}{{.Balance}}{{else}}0{{end}}");
    if (!(balance > 0)) { alert("Invoice already paid."); return; }

    // 1. Create server-side order
    const orderRes = await fetch("/api/v1/payments/razorpay-order", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ invoice_id: invoiceId, amount: balance })
    });
    if (!orderRes.ok) { alert("Could not start Razorpay session."); return; }
    const order = await orderRes.json();

    // 2. Open checkout with the REAL order_id
    const options = {
        key: order.razorpay_key_id,
        order_id: order.order_id,
        amount: order.amount_paise,
        currency: order.currency,
        name: "FlyFleet SaaS",
        description: "Invoice {{.Invoice.InvoiceNumber}} Payment",
        handler: async function (response) {
            // 3. Server-side verify (single source of truth)
            const vRes = await fetch("/api/v1/payments/razorpay/verify", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    invoice_id: invoiceId,
                    razorpay_order_id: response.razorpay_order_id,
                    razorpay_payment_id: response.razorpay_payment_id,
                    razorpay_signature: response.razorpay_signature
                })
            });
            if (!vRes.ok) { alert("Payment verification failed. Contact support."); return; }
            window.location.href = "/invoices/" + invoiceId;
        }
    };
    const rzp = new Razorpay(options);
    rzp.open();
};
</script>
```
The legacy `<form method="POST" action="/payments/new/...">` stays for cash/upi/bank/cheque manual entry; the Razorpay button no longer submits it.

---

## 5. Business logic

### 5.1 Server-side order with notes.invoice_id (fixes broken webhook)
`POST /razorpay-order` handler:
```
tenant := auth.ContextUser.TenantID            // NEVER trust client tenant
inv    := invoices.GetInvoice(ctx, invoiceID)  // 404 if missing
bal    := invoices.GetBalance(ctx, invoiceID)  // from services (same as UI)
if bal <= 0  -> 402 invoice_already_paid
if cfg.RazorpayKeyID == "" || cfg.RazorpayKeySecret == "" -> 503
order  := razorpayClient.CreateOrder(invoiceID, bal, "INR")   // sets notes.invoice_id
// persist a PENDING payment row keyed by order id for reconcile (optional but recommended)
return { order_id, razorpay_key_id, amount_paise: bal*100, currency:"INR", invoice_id }
```
Now `notes.invoice_id` flows into every downstream webhook (`payment.captured`, `order.paid`), so `razorpay_webhook.go:232-256` will no longer hit `ErrWebhookInvoiceMissing`.

### 5.2 Verify = single source of truth
`POST /razorpay/verify`:
```
ok := razorpayClient.VerifyPaymentSignature(order_id, payment_id, signature)
if !ok -> 401 invalid_signature
existing := repo.FindByRazorpayPaymentID(tenant, payment_id)
if existing != "" -> return existing (200)        // idempotent
id := recordUC.Execute(Method=razorpay, Reference=&payment_id, ...)
repo.SetRazorpayFields(id, order_id, payment_id, signature)
mark invoice paid/partial via invoice service
```
`razorpay_client.go:78` is the ONLY verifier used here. The browser never asserts success.

### 5.3 Idempotent, restart-safe webhook
The `razorpay_payment_id` layer is ALREADY restart-safe via `reference`→`idempotency_key` (UNIQUE `idx_payments_idempotency`, `00035`). Only the `eventID` layer relies on the in-memory `processedIDs` map (`razorpay_webhook.go:103-123`) and is lost on restart. Add a DB-backed `webhook_event_id` column so event-id dedup also survives restart, keeping the map as a hot cache:
```
ExecuteEvent(ctx, body, sig, eventID, ev):
  if err := VerifySignature(body, sig); err != nil -> err
  // DB dedup by webhook_event_id
  if eventID != "":
     if repo.ExistsWebhookEvent(tenant, eventID) -> return existingPaymentID (already recorded)
  // DB dedup by razorpay_payment_id (covers verify-then-webhook race)
  pid := ev.Payload.Payment.Entity.ID  // or Order.ID / Refund.PaymentID
  if pid != "" and repo.ExistsRazorpayPayment(tenant, pid):
     return existingPaymentID
  ... apply side effect (record / reverse / emit) ...
  repo.SetWebhookEventID(paymentID, eventID)   // persist for restart safety
  // keep in-memory map as hot cache
```
Persistence means a process restart loses the map but the DB still blocks duplicates → no double count.

### 5.4 Mark invoice paid
After a successful record (verify OR webhook), call the existing invoice settlement path used by `internal/handlers/payments.go:135-143` (record then redirect to `/invoices/{id}`). Recompute invoice status from sum(payments) vs total.

### 5.5 Refunds
`refund.processed` webhook already calls `reverseUC.Execute` (`razorpay_webhook.go:284-308`). Ensure `ReversePaymentUseCase` also stamps `razorpay_payment_id`/notes so a refund is traceable. Refund amount may be partial — keep the original payment row; create a reversing (negative) payment row.

---

## 6. Config / env

| Var | Default | Purpose | Read by |
|-----|---------|---------|---------|
| `RAZORPAY_KEY_ID` | `""` | Publishable key sent to browser + order creation | `internal/config/config.go:119` → `cfg.RazorpayKeyID` |
| `RAZORPAY_KEY_SECRET` | `""` | Server-side order creation + signature verification | `internal/config/config.go:120` → `cfg.RazorpayKeySecret` |
| `RAZORPAY_WEBHOOK_SECRET` | `""` | HMAC verify of inbound webhooks | `internal/config/config.go:121` → `cfg.RazorpayWebhook` |

All three already exist in `config.go`. No code change required in config — only ensure the new order/verify handlers read `cfg.RazorpayKeyID/KeySecret` (pass `razorpayClient` into the handler, constructed from these at `cmd/server/main.go:257` area). When any are empty, endpoints return 503 so the app boots and the rest works without Razorpay creds (mock-friendly per template conventions).

---

## 7. Tests

Coverage gate: new code ≥ 80%. Run `go test ./internal/payment/... ./internal/handlers/...`.

### 7.1 Known-vector signature test (checkout verify)
Razorpay's documented test vector: with secret `1234567890abcdef`, `order_id = 1`, `payment_id = 2`, the expected signature equals HMAC-SHA256(`"1|2"`, secret) hex. Add to `internal/payment/razorpay/client_test.go`:
```go
func TestVerifyPaymentSignature_KnownVector(t *testing.T) {
    c := NewRazorpayClient("key", "1234567890abcdef")
    mac := hmac.New(sha256.New, []byte("1234567890abcdef"))
    mac.Write([]byte("1|2"))
    want := hex.EncodeToString(mac.Sum(nil))
    if !c.VerifyPaymentSignature("1", "2", want) {
        t.Fatal("known vector should verify")
    }
    if c.VerifyPaymentSignature("1", "2", "deadbeef") {
        t.Fatal("wrong sig must fail")
    }
}
```

### 7.2 Order sets invoice note
Unit test `CreateOrder` (use a fake `rzp.Client` or assert the `notes.invoice_id` map passed in). If the real SDK call is undesirable in unit tests, inject an interface:
```go
type OrderCreator interface { CreateOrder(invoiceID string, amt float64, cur string) (*Order, error) }
```
Assert `CreateOrder("inv_123", 10, "INR")` produces `notes.invoice_id == "inv_123"` and `amount == 1000` paise.

### 7.3 Webhook idempotent across restart
In `razorpay_webhook_test.go`, feed the SAME `payment.captured` body twice. First call records; after simulating restart (new `RazorpayWebhookUseCase` instance, fresh in-memory map), second call with same `eventID` + `razorpay_payment_id` returns the SAME `paymentID` and records exactly ONE row (assert `COUNT(*) == 1` in DB).

### 7.4 Verify-then-webhook no double count
1. `POST /razorpay/verify` with valid signature → records payment `pay_1`.
2. Deliver `payment.captured` webhook with `razorpay_payment_id = pay_1` and `notes.invoice_id` set.
3. Assert exactly ONE payment row for `pay_1` and invoice balance reduced once.

### 7.5 Pass-before-merge checklist
- [ ] `go build ./...` clean
- [ ] `go vet ./internal/payment/...`
- [ ] signature known-vector test passes
- [ ] order note test passes
- [ ] webhook restart idempotency test passes
- [ ] verify-then-webhook test passes
- [ ] `goose up` 00057 applies; `goose down` reverts cleanly
- [ ] manual: UI button creates order (DevTools shows `order_id` in checkout options), verify records `method=razorpay`, webhook in dashboard shows `payment.captured` applied (not dropped)

---

## 8. Future / GPS-provider

- **Toll / FASTag auto-collect**: reuse the same `webhook_event_id`/`razorpay_payment_id` idempotency pattern for aggregator settlement webhooks; pair with `00049 FASTag tables`.
- **UPI QR auto-pay**: generate a static/dynamic UPI QR linked to `order_id`; on scan-pay, the `payment.captured` webhook (now working) closes the loop with no extra UI.
- **GPS / telematics**: out of scope here; follow the `TelematicsProvider` interface convention from the template if toll collection needs vehicle-geofence correlation.

---

## 9. Edge cases

| Case | Handling |
|------|----------|
| Webhook delivered before /verify completes (race) | DB unique `idx_payments_razorpay_payment` blocks the second writer; both return same id |
| `X-Razorpay-Event-Id` header absent | fall back to dedup by `razorpay_payment_id`; ack 200 so Razorpay stops retrying |
| `notes.invoice_id` missing in webhook | ack 200 (Razorpay retries only on non-2xx), log + emit `RazorpayPaymentFailed`, do NOT 4xx |
| Invalid checkout signature | `/verify` returns 401; no payment recorded; UI alerts user |
| Partial refund | keep original + add reversing (negative) row; do not delete original |
| Razorpay creds empty | order/verify return 503; manual cash/upi still works; webhook returns 503 (configured check) |
| Double-click Pay button | order endpoint idempotent per invoice; restart-safe map + DB dedup |
| Amount mismatch (client sends wrong amount) | server ignores client amount, always uses invoice balance from `GetBalance` |
| Tenant isolation | always derive tenant from `auth.ContextUser`; never accept client `tenant_id` |

---

## 10. Phased rollout

1. **Migration 00057** + legacy enum align (entity.go, aliases.go). `goose up`.
2. **Wire dead client**: construct `razorpayClient` in `cmd/server/main.go` from `cfg`; inject into a new order/verify handler.
3. **Add endpoints** `/razorpay-order` and `/razorpay/verify` in the protected group; reuse `RecordPaymentUseCase`.
4. **Persist webhook idempotency** in `razorpay_webhook.go` (DB-backed `webhook_event_id` + `razorpay_payment_id`), keep in-memory cache.
5. **UI swap** in `payment_edit.html` (order→verify→reload).
6. **Tests** (§7) + manual smoke.

---

## 11. Open items / VERIFY

- **Verify endpoint tenant/permission**: confirm `payments:create` is the right RBAC action for both order + verify (currently `Record` uses `payments:create` at `payment_handler.go:59`). ✅ consistent.
- **Pending vs recorded**: decide if `/razorpay-order` writes a PENDING row or only creates the Razorpay order. Spec recommends creating only the Razorpay order and recording at `/verify` (simpler, no orphan rows). VERIFY with lead.
- **Invoice status recompute**: confirm the invoice service used by `internal/handlers/payments.go:135-143` is the canonical "mark paid" path and reuse it (avoid a second implementation).
- **`razorpay_client.go` SDK call in tests**: confirm we can fake `rzp.Client` or wrap `OrderCreator` so unit tests need no network.
- **Webhook secret rotation**: ensure `cfg.RazorpayWebhook` reload path exists if secrets rotate at runtime (currently loaded once at boot — acceptable for v1).

---

## 12. File list

Create:
- `db/migrations/00057_razorpay_payment_fields.sql` — columns + 2 unique partial indexes (Up/Down).
- `internal/payment/razorpay/client_test.go` — known-vector signature test.
- `internal/payment/application/razorpay_webhook_test.go` — restart idempotency + verify-then-webhook tests.
- `internal/payment/presentation/api/handlers/razorpay_order_handler.go` — `RazorpayOrder` + `RazorpayVerify` handlers (or extend `payment_handler.go`).

Modify:
- `internal/domain/payment/entity.go` — add `PaymentMethodRazorpay` (align enum).
- `internal/domain/aliases.go` — add `PaymentMethodRazorpay` alias.
- `internal/payment/presentation/api/dto/payment_dto.go` — add `RazorpayOrderID/PaymentID/Signature/WebhookEventID`.
- `internal/payment/infrastructure/persistence/sql/converters/payment_converter.go` — map new columns.
- `internal/payment/application/razorpay_webhook.go` — DB-backed idempotency (`webhook_event_id`, `razorpay_payment_id`).
- `internal/templates/payment_edit.html` — add razorpay `<option>`; replace button JS with order→verify→reload.
- `cmd/server/main.go` — construct `razorpayClient`; mount order/verify routes in protected group (near `:257`, `:443-455`).
- `internal/payment/presentation/api/handlers/payment_handler.go` — register new routes + wire `razorpayClient`.

Do NOT touch:
- `db/migrations/00042` (`company_config` owner).
- `00037`/`00035` (already correct).
- `internal/payment/razorpay/client.go` public signatures (`CreateOrder`, `VerifyPaymentSignature`) — keep, just wire them.
