package nullchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var ErrInvalidRequest = errors.New("invalid request")

func (n *NullBlockchain) Stop() {
	if n.srv == nil {
		return
	}

	n.log.Info("Stopping nullchain server")
	if err := n.srv.Shutdown(context.Background()); err != nil {
		n.log.Warn("failed to shutdown")
	}
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

	n.srv = &http.Server{Addr: n.srvAddress}
	http.HandleFunc("/api/v1/forwardtime", n.handleForwardTime)

	n.log.Info("starting backdoor server")
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
