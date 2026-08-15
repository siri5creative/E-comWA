package util

import "testing"

func TestNormalizeWhatsAppNumber(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"leading zero converted to 62", "081234567890", "6281234567890", false},
		{"already 62-prefixed kept as-is", "6281234567890", "6281234567890", false},
		{"leading plus stripped then 62 recognized", "+6281234567890", "6281234567890", false},
		{"spaces and dashes stripped", "0812-3456-7890", "6281234567890", false},
		{"spaces around number trimmed", "  081234567890  ", "6281234567890", false},

		{"no recognizable prefix rejected", "81234567890", "", true},
		{"contains letters rejected", "0812abc7890", "", true},
		{"too short rejected", "0812345", "", true},
		{"too long rejected", "0812345678901234567", "", true},
		{"empty string rejected", "", "", true},
		{"just a plus sign rejected", "+", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeWhatsAppNumber(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizeWhatsAppNumber(%q) = %q, nil; want an error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeWhatsAppNumber(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeWhatsAppNumber(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}
