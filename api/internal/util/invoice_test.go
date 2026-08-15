package util

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateInvoiceNumber(t *testing.T) {
	got, err := GenerateInvoiceNumber("INV")
	if err != nil {
		t.Fatalf("GenerateInvoiceNumber returned error: %v", err)
	}

	wantDatePrefix := "INV-" + time.Now().UTC().Format("20060102") + "-"
	if !strings.HasPrefix(got, wantDatePrefix) {
		t.Fatalf("GenerateInvoiceNumber() = %q; want prefix %q", got, wantDatePrefix)
	}

	suffix := strings.TrimPrefix(got, wantDatePrefix)
	if len(suffix) != 6 {
		t.Fatalf("GenerateInvoiceNumber() suffix = %q, len %d; want len 6", suffix, len(suffix))
	}
	for _, c := range suffix {
		if !strings.ContainsRune(invoiceRandomAlphabet, c) {
			t.Fatalf("GenerateInvoiceNumber() suffix %q contains unexpected character %q", suffix, c)
		}
	}
}

func TestGenerateInvoiceNumberUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	const n = 200
	for i := 0; i < n; i++ {
		got, err := GenerateInvoiceNumber("POS")
		if err != nil {
			t.Fatalf("GenerateInvoiceNumber returned error: %v", err)
		}
		if seen[got] {
			t.Fatalf("GenerateInvoiceNumber produced a duplicate after %d calls: %q", i, got)
		}
		seen[got] = true
	}
}
