// Copyright (C) 2023  Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
