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

const (
	NullChainStatusReady     = "chain-ready"
	NullChainStatusReplaying = "chain-replaying"
)

func (n *NullBlockchain) Stop() error {
	if n.replayer != nil {
		if err := n.replayer.Stop(); err != nil {
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
	n.srv = &http.Server{Addr: net.JoinHostPort(n.cfg.IP, strconv.Itoa(n.cfg.Port))}
	http.HandleFunc("/api/v1/forwardtime", n.handleForwardTime)
	http.HandleFunc("/api/v1/status", n.status)

	n.log.Info("starting time-forwarding server", logging.String("addr", n.srv.Addr))
	go n.srv.ListenAndServe()

	n.log.Info("starting blockchain")
	if err := n.StartChain(); err != nil {
		return err
	}

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

	if n.replaying.Load() {
		http.Error(w, ErrChainReplaying.Error(), http.StatusServiceUnavailable)
	}

	// we need to call ForwardTime in a different routine so that if it panics it stops the node instead of just being
	// caught in the http-handler recover. But awkwardly it seems like the vega-sim relies on the http-request not
	// returning until the time-forward has finished, so we need to preserve that for now.
	done := make(chan struct{})
	go func() {
		n.ForwardTime(d)
		done <- struct{}{}
	}()
	<-done
}

// status returns the status of the nullchain, whether it is replaying or whether its ready to go.
func (n *NullBlockchain) status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := struct {
		Status string `json:"status"`
	}{
		Status: NullChainStatusReady,
	}

	if n.replaying.Load() {
		resp.Status = NullChainStatusReplaying
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
