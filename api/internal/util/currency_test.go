package util

import "testing"

func TestFormatRupiah(t *testing.T) {
	cases := []struct {
		amount int64
		want   string
	}{
		{0, "Rp0"},
		{500, "Rp500"},
		{1000, "Rp1.000"},
		{85000, "Rp85.000"},
		{170000, "Rp170.000"},
		{1000000, "Rp1.000.000"},
		{999, "Rp999"},
		{-15000, "-Rp15.000"},
	}

	for _, tc := range cases {
		got := FormatRupiah(tc.amount)
		if got != tc.want {
			t.Errorf("FormatRupiah(%d) = %q; want %q", tc.amount, got, tc.want)
		}
	}
}
