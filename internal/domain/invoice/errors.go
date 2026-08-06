package invoice

import "errors"

var (
	ErrInvoiceNotFound  = errors.New("invoice not found")
	ErrDuplicateInvoice = errors.New("invoice already exists for this trip")
)
