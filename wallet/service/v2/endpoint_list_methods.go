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

package v2

import (
	"encoding/json"
	"net/http"
	"sort"

	vfmt "code.vegaprotocol.io/vega/libs/fmt"
	"code.vegaprotocol.io/vega/logging"
	"github.com/julienschmidt/httprouter"
)

type ListMethodsResponse struct {
	RegisteredMethods []string `json:"registeredMethods"`
}

func (a *API) ListMethods(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	a.log.Info("New request",
		logging.String("url", vfmt.Escape(r.URL.String())),
	)

	registeredMethods := make([]string, 0, len(a.commands))
	for method := range a.commands {
		registeredMethods = append(registeredMethods, method)
	}

	sort.Strings(registeredMethods)

	body, _ := json.Marshal(Response{
		Result: ListMethodsResponse{
			RegisteredMethods: registeredMethods,
		},
	})

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		a.log.Info("Internal error",
			logging.Int("http-status", http.StatusInternalServerError),
			logging.Error(err),
		)
		return
	}

	a.log.Info("Success",
		logging.Int("http-status", http.StatusOK),
	)
}
