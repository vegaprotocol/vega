// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nullchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

var ErrInvalidRequest = errors.New("invalid request")

func (n *NullBlockchain) Stop() error {
	if r, ok := n.app.(*Replayer); ok {
		if err := r.Stop(); err != nil {
			n.log.Error("failed to stop nullchain replayer")
		}
	}

	if n.srv == nil {
		return nil
	}

	n.log.Info("Stopping nullchain server")
	if err := n.srv.Shutdown(context.Background()); err != nil {
		n.log.Error("failed to shutdown")
	}

	return nil
}

func (n *NullBlockchain) Start() error {
	// the nullblockchain needs to start after the grpc API have started, so we pretend to start here
	return nil
}

func (n *NullBlockchain) StartServer() error {
	n.log.Info("starting blockchain")
	if err := n.StartChain(); err != nil {
		return err
	}

	n.srv = &http.Server{Addr: net.JoinHostPort(n.cfg.IP, strconv.Itoa(n.cfg.Port))}
	http.HandleFunc("/api/v1/forwardtime", n.handleForwardTime)

	n.log.Info("starting time-forwarding server", logging.String("addr", n.srv.Addr))
	go n.srv.ListenAndServe()
	return nil
}

// RequestToDuration should receive either be a parsable duration or a RFC3339 datetime.
func RequestToDuration(req string, now time.Time) (time.Duration, error) {
	d, err := time.ParseDuration(req)
	if err != nil {
		newTime, err := time.Parse(time.RFC3339, req)
		if err != nil {
			return 0, fmt.Errorf("%w: time is not a duration or RFC3339 datetime", ErrInvalidRequest)
		}

		// Convert to a duration by subtracting the current frozen time of the nullchain
		d = newTime.Sub(now)
	}

	if d < 0 {
		return 0, fmt.Errorf("%w: cannot step backwards in time", ErrInvalidRequest)
	}

	return d, nil
}

// handleForwardTime processes the incoming request to shuffle forward in time converting it into a valid
// duration from `now` and pushing it into the nullchain to do its thing.
func (n *NullBlockchain) handleForwardTime(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unexpected request body", http.StatusBadRequest)
		return
	}
	req := struct {
		Forward string `json:"forward"`
	}{}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "unexpected request body", http.StatusBadRequest)
		return
	}

	d, err := RequestToDuration(req.Forward, n.now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// Do the dance
	n.ForwardTime(d)
}
