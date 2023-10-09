// Copyright (C) 2023 Gobalsky Labs Limited
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

package gateway

import (
	"crypto/tls"
	"errors"
	"net/http"

	"code.vegaprotocol.io/vega/paths"
	"golang.org/x/crypto/acme/autocert"
)

func GenerateTlsConfig(g *Config, vegaPaths paths.Paths) (*tls.Config, http.Handler, error) {
	if g.HTTPSEnabled {
		if g.AutoCertDomain != "" {
			if g.CertificateFile != "" || g.KeyFile != "" {
				return nil, nil, errors.New("autocert is enabled, and a pre-generated certificate/key specified; use one or the other")
			}
			certDir := vegaPaths.StatePathFor(paths.DataNodeAutoCertHome)

			certManager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(g.AutoCertDomain),
				Cache:      autocert.DirCache(certDir),
			}

			return &tls.Config{
				GetCertificate: certManager.GetCertificate,
				// NextProtos:     []string{"http/1.1", "acme-tls/1"},
			}, certManager.HTTPHandler(nil), nil
		}

		certificate, err := tls.LoadX509KeyPair(g.CertificateFile, g.KeyFile)
		if err != nil {
			return nil, nil, err
		}
		certificates := []tls.Certificate{certificate}
		return &tls.Config{
			Certificates: certificates,
		}, nil, nil
	}

	return nil, nil, nil
}
