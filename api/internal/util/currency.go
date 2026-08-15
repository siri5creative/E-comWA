package util

import "strconv"

// FormatRupiah renders whole-Rupiah amounts as "Rp170.000" for WhatsApp
// message text (plain ASCII — WA renders it as-is, no locale formatting
// available server-side).
func FormatRupiah(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}

	digits := strconv.FormatInt(amount, 10)
	var grouped []byte
	for i, d := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped = append(grouped, '.')
		}
		grouped = append(grouped, d)
	}

	if negative {
		return "-Rp" + string(grouped)
	}
	return "Rp" + string(grouped)
}
