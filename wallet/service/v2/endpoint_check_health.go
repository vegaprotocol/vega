package v2

import (
	"net/http"

	vfmt "code.vegaprotocol.io/vega/libs/fmt"
	"code.vegaprotocol.io/vega/logging"
	"github.com/julienschmidt/httprouter"
)

func (a *API) CheckHealth(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	a.log.Debug("New request",
		logging.String("url", vfmt.Escape(r.URL.String())),
	)

	w.WriteHeader(http.StatusOK)
}
