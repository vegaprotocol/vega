package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/auth/handler"
	"code.vegaprotocol.io/vega/logging"
)

type Callback func([]PartyInfo)

type PartyInfo struct {
	ID       string `json:"id"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

type Svc struct {
	log *logging.Logger
	Config
	ctx context.Context

	client    *http.Client
	mu        sync.Mutex
	parties   []PartyInfo
	listeners []Callback
}

func New(ctx context.Context, log *logging.Logger, cfg Config) (*Svc, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	// create httpclient
	client := http.Client{
		Timeout: cfg.Timeout.Get(),
	}

	s := &Svc{
		log:       log,
		Config:    cfg,
		ctx:       ctx,
		client:    &client,
		parties:   []PartyInfo{},
		listeners: []Callback{},
	}

	// try to reach the serv once first, so as soon as other
	// services are starting we have access to the auth list.
	s.update()
	go s.start()
	return s, nil
}

func (s *Svc) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.mu.Lock()
	s.Config = cfg
	s.mu.Unlock()
}

func (s *Svc) update() bool {
	s.log.Debug("updating list of authorized parties")
	s.mu.Lock()
	defer s.mu.Unlock()

	req, _ := http.NewRequest(http.MethodGet, s.ServerAddr, nil)
	req = addToken(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Warn("unable to call authentication service",
			logging.Error(err),
		)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.log.Warn("unable to read body from response",
			logging.Error(err),
		)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		s.log.Warn("received error from auth server",
			logging.String("error", string(body)),
		)
		return false
	}

	payload := struct {
		Parties []PartyInfo `json:"parties"`
	}{}
	err = json.Unmarshal(body, &payload)
	if err != nil {
		s.log.Warn("unable to read body from response",
			logging.Error(err),
		)
		return false
	}

	s.parties = payload.Parties
	s.log.Debug("list of parties updated",
		logging.Reflect("parties", s.parties),
	)

	return true
}

func (s *Svc) Get() []PartyInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PartyInfo, 0, len(s.parties))
	out = append(out, s.parties...)
	return out
}

func (s *Svc) OnPartiesUpdated(f Callback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, f)
}

func (s *Svc) notify() {
	parties := s.Get()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.listeners {
		go f(parties)
	}
}

func (s *Svc) start() {
	ticker := time.NewTicker(s.Interval.Get())
	for {
		select {
		case <-ticker.C:
			ok := s.update()
			if ok {
				s.notify()
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func addToken(r *http.Request) *http.Request {
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %v", handler.Token))
	return r
}
