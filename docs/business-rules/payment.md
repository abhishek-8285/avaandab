# Payment Business Rules

## Payment Methods

| Method          | Code           |
|-----------------|----------------|
| Cash            | `cash`         |
| UPI             | `upi`          |
| Bank Transfer   | `bank_transfer`|
| Cheque          | `cheque`       |

## Rules

### Recording a Payment
* A payment must reference a valid `InvoiceID`.
* `Amount` must be greater than zero.
* `PaymentMethod` must be one of the four accepted methods above.
* `PaymentDate` defaults to the current timestamp.
* A payment can be **partial** — it does not need to cover the full outstanding balance.

### Outstanding Balance Update
* Each payment reduces the invoice's `Outstanding` balance by the payment amount.
* If `Outstanding` becomes 0, the invoice status is updated to `Paid`.
* If `Overpayment` occurs (payment amount exceeds outstanding), the excess is recorded as `CreditBalance` on the invoice.

### Payment History
* All payments for an invoice are listed chronologically.
* The `Amount` is always positive (refunds are handled separately).

### Immutability
* Once recorded, a payment cannot be edited or deleted.
* If a payment was entered in error, create a credit/reversal payment entry.

### Invoice Status Sync
* When a payment is recorded, the parent invoice's `PaymentStatus` is automatically updated:
  * `Outstanding == 0` → `Paid`
  * `Outstanding > 0 && AmountPaid > 0` → `Partially Paid`
  * `AmountPaid == 0` → `Pending`

### Permissions
| Action   | Permission        |
|----------|-------------------|
| Record   | `payments:create`  |
| Read     | `payments:read`   |
| Void     | `payments:delete` |
