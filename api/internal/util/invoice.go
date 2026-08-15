package util

import (
	"crypto/rand"
	"fmt"
	"time"
)

const invoiceRandomAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateInvoiceNumber builds a human-readable, effectively-unique invoice
// number like "INV-20260815-A1B2C3". The random suffix (36^6 combinations
// per day) makes a collision astronomically unlikely, so callers don't need
// retry logic — the orders.invoice_number UNIQUE constraint remains the
// backstop of last resort.
func GenerateInvoiceNumber(prefix string) (string, error) {
	raw := make([]byte, 6)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate invoice suffix: %w", err)
	}

	suffix := make([]byte, len(raw))
	for i, b := range raw {
		suffix[i] = invoiceRandomAlphabet[int(b)%len(invoiceRandomAlphabet)]
	}

	return fmt.Sprintf("%s-%s-%s", prefix, time.Now().UTC().Format("20060102"), suffix), nil
}
