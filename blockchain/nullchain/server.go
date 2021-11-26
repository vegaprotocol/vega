package nullchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var ErrInvalidRequest = errors.New("invalid request")

func (n *NullBlockchain) StartServer() error {
	http.HandleFunc("/forwardtime", n.handleForwardTime)

	// Fire up http server
	go http.ListenAndServe(n.srvAddress, nil)

	return nil
}

// requestToDuration req should either be a parsable duration or a RFC3339 datetime
// work out which.
func requestToDuration(req string, now time.Time) (time.Duration, error) {
	d, err := time.ParseDuration(req)
	if err == nil {
		return d, nil
	}

	newTime, err := time.Parse(time.RFC3339, req)
	if err != nil {
		return 0, fmt.Errorf("%w: time is not a duration or RFC3339 datetime", ErrInvalidRequest)
	}

	// Convert to a duration by subtracting the current frozen time of the nullchain
	d = newTime.Sub(now)
	if d < 0 {
		return 0, fmt.Errorf("%w: cannot step backwards in time %s < %s", ErrInvalidRequest, newTime, now)
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

	d, err := requestToDuration(req.Forward, n.now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// Do the dance
	n.ForwardTime(d)
	w.WriteHeader(http.StatusOK)
}
