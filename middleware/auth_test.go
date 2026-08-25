package middleware

import "testing"

func TestIsGeminiAPIKeyPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/v1/models", want: true},
		{path: "/v1/models/gemini-2.5-pro", want: true},
		{path: "/v1beta/models", want: true},
		{path: "/v1beta/openai/models", want: true},
		{path: "/v1/chat/completions", want: false},
		{path: "/v1/models-extra", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isGeminiAPIKeyPath(tt.path); got != tt.want {
				t.Fatalf("isGeminiAPIKeyPath(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}
