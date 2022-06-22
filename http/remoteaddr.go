// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package http

import (
	"fmt"
	"net"
	"net/http"
)

// RemoteAddr collects possible remote addresses from a request
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
