package utils

import "testing"

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		acceptLang  string
		defaultLang string
		want        string
	}{
		{"en-US,en;q=0.9,fr;q=0.8", "en", "en"},
		{"fr-FR", "en", "fr"},
		{"es", "en", "es"},
		{"en-GB;q=0.9,en;q=0.8", "en", "en"},
		{"", "en", "en"},
		{"en-US", "fr", "en"},
		{"zh-CN,zh;q=0.9", "en", "zh"},
	}

	for _, tt := range tests {
		got := DetectLanguage(tt.acceptLang, tt.defaultLang)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q, %q) = %q, want %q", tt.acceptLang, tt.defaultLang, got, tt.want)
		}
	}
}
