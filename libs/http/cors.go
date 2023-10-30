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

import (
	"net/http"
	"strings"

	"github.com/rs/cors"
)

// CORSConfig represents the configuration for CORS.
type CORSConfig struct {
	AllowedOrigins []string `description:"Allowed origins for CORS"                 long:"allowed-origins"`
	MaxAge         int      `description:"Max age (in seconds) for preflight cache" long:"max-age"`
}

func CORSOptions(config CORSConfig) cors.Options {
	return cors.Options{
		AllowOriginFunc: AllowedOrigin(config.AllowedOrigins),
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		MaxAge:           config.MaxAge,
		AllowCredentials: false,
	}
}

func AllowedOrigin(allowedOrigins []string) func(origin string) bool {
	trimScheme := func(origin string) string {
		return strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
	}
	return func(origin string) bool {
		if len(allowedOrigins) == 0 || allowedOrigins[0] == "*" {
			return true
		}
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == origin || trimScheme(allowedOrigin) == trimScheme(origin) {
				return true
			}
		}
		return false
	}
}
