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
	"fmt"
	"net"
	"net/http"
)

// RemoteAddr collects possible remote addresses from a request.
func RemoteAddr(r *http.Request) (string, error) {
	// Only defined when site is accessed via non-anonymous proxy
	// and takes precedence over RemoteAddr
	remote := r.Header.Get("X-Forwarded-For")
	if remote != "" {
		return remote, nil
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("unable to get remote address (failed to split host:port) from \"%s\": %v", r.RemoteAddr, err)
	}

	return ip, nil
}
