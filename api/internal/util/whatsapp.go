package util

import (
	"errors"
	"regexp"
	"strings"
)

var ErrInvalidWhatsAppNumber = errors.New("nomor WhatsApp tidak valid")

var digitsOnly = regexp.MustCompile(`^[0-9]+$`)

// NormalizeWhatsAppNumber converts a customer-entered WhatsApp number into
// the international 62xxx format required by wa.me links — no leading "+",
// "00", or "0" (PRD section 11). A leading "0" is replaced with "62"; a
// number already starting with "62" is kept as-is. Any other shape is
// rejected rather than guessed at.
func NormalizeWhatsAppNumber(raw string) (string, error) {
	n := strings.TrimSpace(raw)
	n = strings.ReplaceAll(n, " ", "")
	n = strings.ReplaceAll(n, "-", "")
	n = strings.TrimPrefix(n, "+")

	switch {
	case strings.HasPrefix(n, "0"):
		n = "62" + strings.TrimPrefix(n, "0")
	case strings.HasPrefix(n, "62"):
		// already in the expected format
	default:
		return "", ErrInvalidWhatsAppNumber
	}

	if !digitsOnly.MatchString(n) || len(n) < 10 || len(n) > 15 {
		return "", ErrInvalidWhatsAppNumber
	}

	return n, nil
}
