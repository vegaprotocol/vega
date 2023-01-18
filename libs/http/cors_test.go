package http

import "testing"

func TestAllowedOrigin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		want           bool
	}{
		{
			name:           "no allowed origins",
			allowedOrigins: nil,
			origin:         "http://example.com",
			want:           true,
		}, {
			name:           "empty allowed origins",
			allowedOrigins: []string{},
			origin:         "http://example.com",
			want:           true,
		}, {
			name:           "wildcard",
			allowedOrigins: []string{"*"},
			origin:         "http://example.com",
			want:           true,
		}, {
			name:           "match",
			allowedOrigins: []string{"http://example.com"},
			origin:         "http://example.com",
			want:           true,
		}, {
			name:           "match with scheme",
			allowedOrigins: []string{"http://example.com"},
			origin:         "https://example.com",
			want:           true,
		}, {
			name:           "no match",
			allowedOrigins: []string{"http://example.com"},
			origin:         "http://not-example.com",
			want:           false,
		}, {
			name:           "match with scheme",
			allowedOrigins: []string{"https://example.com"},
			origin:         "http://example.com",
			want:           true,
		}, {
			name:           "no match with scheme",
			allowedOrigins: []string{"https://example.com"},
			origin:         "http://not-example.com",
			want:           false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := AllowedOrigin(tt.allowedOrigins)(tt.origin); got != tt.want {
				t.Errorf("AllowedOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}
