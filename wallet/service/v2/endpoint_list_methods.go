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
