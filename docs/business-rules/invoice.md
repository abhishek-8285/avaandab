# Invoice Business Rules

## Status Lifecycle

```text
Pending → Partially Paid → Paid
   ↓         ↓
  Voided    -
```

## Rules

### Generation
* Invoices are generated automatically upon trip completion.
* An invoice is created for each completed trip.
* **Idempotent**: generating an invoice for the same booking twice returns the same invoice (no duplicates).
* The `BookingID` and `CustomerID` must be provided.
* `Subtotal`, `Tax`, `Discount`, and `Total` must all be non-negative.
* `Total` must equal `Subtotal + Tax — Discount`.

### Invoice Numbering
* Each invoice receives a unique, sequential invoice number (`INV-0001` format).
* The numbering is scoped per tenant (if multi-tenancy is enabled).

### GST Calculation
* `Tax` is calculated as `Subtotal × GST_rate`.
* The default GST rate is 12% (configurable per tenant).
* `Total = Subtotal + Tax — Discount`.

### Outstanding Balance
* `Outstanding = Total — AmountPaid`
* `AmountPaid` starts at 0 and increases with each payment.
* When `Outstanding <= 0`, status becomes `Paid`.

### Payment Status
* `Pending`: no payment received.
* `Partially Paid`: at least one payment received, but `Outstanding > 0`.
* `Paid`: `Outstanding <= 0`.
* `Voided`: invoice was cancelled/abandoned (manual action only).

### Cancellation / Voiding
* Only `Pending` invoices can be voided.
* Once `Paid`, an invoice cannot be voided (only refunded).
