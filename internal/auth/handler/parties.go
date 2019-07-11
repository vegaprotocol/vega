package handler

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const (
	Token = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1ODk5ODQxMjksImlkIjoiYjgxNjk4ZWI1YWY5ZDZmNTMwZmFjOTEwYmMwYWRkYTcwYzQ5YzE3NGFmOWY4ZjU5NmIzODExODk3NDRjNmNiOCJ9.tCU56ntqbX2Mr8b-bYW-Re59WFZU4rE_9blL6nOBzme6OaQWUxUBYc39FeCOZBQGGao8MvyjLY8s4qd6sxVoxyA75qToAIAyRsDblzwdokRTsH7tkeFix0mUmLJN2kMJx8bVVFjCesXenZ4j_lF26BxvYByQi-UGcdl4EdpxbwtrhdILXFel5WDA97gizVWrBG2qOGdA7vujHeuE5CTWRhIXj5REIGtbLNoXpEaN_c7pWsQ4l-xxsfFqAhbuD3lQTMyLZvStAyy9aqDymZskF_4CKGAV9ASs_S9TVc_olhPfGNjDaj1ku71NEait7ZAEQd8HxKnZiLb22_iI6jxyiIAHEQq1cCT_mn8dAW3rU2J2Js_geJWAaRWhLdJ3gnK5fqI8MEvJm03ZgiqOQ0IEW9sVQOc2uLNu9FPTbzH0Lbkk8kgCu68ZgfWw24A-oEvWREfpMxZrLDEJ2nNsJ7PNarE0wxEFNfaW5Z3YYg_k3CSReYGk3tQEjrEKH3tqExusd-SYOnCuHhZmwPxtrgsGlxpToKBXm_JxxI0G3VITIEAk95El6nAXcZxqPV-G4tE_rGvo6Z_60cc0DKco0Ut323YIIrc4FvDp1GroFBmV1paArUx1a5apTSjTqSAJ8Ox_LGMrLEOOmxYRTFkwvvFTk5FEj1a303HTzg_u5v8Er78"
)

type Party struct {
	ID    string `json:"id"`
	Pass  string `json:"password"`
	Token string `json:"token"`
}

type Config struct {
	Parties []Party `json:"parties"`
}

type PartyService struct {
	File string
	cfg  Config
	mu   sync.Mutex
}

func (p *PartyService) Load() error {
	p.mu.Lock()
	buf, err := ioutil.ReadFile(filepath.Join(p.File))
	if err != nil {
		p.mu.Unlock()
		return err
	}

	cfg := Config{}
	err = json.Unmarshal(buf, &cfg)
	if err != nil {
		p.mu.Unlock()
		return err
	}

	p.cfg = cfg
	p.mu.Unlock()
	// rewrite file now
	return p.updatefile()
}

func (p *PartyService) updatefile() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	buf, err := json.Marshal(&p.cfg)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(p.File, buf, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (p *PartyService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		p.getParties(w, r)
	case http.MethodPost:
		p.addParty(w, r)
	}
}

func (p *PartyService) getParties(w http.ResponseWriter, r *http.Request) {
	if err := checkToken(r); err != nil {
		log.Printf("token error: %v", err)
		http.Error(w, "token error", http.StatusBadRequest)
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	buf, err := json.Marshal(p.cfg)
	if err != nil {
		log.Printf("error marshaling parties: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(buf)
}

func (p *PartyService) addParty(w http.ResponseWriter, r *http.Request) {
	if err := checkToken(r); err != nil {
		log.Printf("token error: %v", err)
		http.Error(w, "token error", http.StatusBadRequest)
		return
	}

	payload := Party{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		log.Printf("error unmarshalling request: %v", err)
		http.Error(w, "can't unmarshal request", http.StatusBadRequest)
		return
	}

	if len(payload.ID) <= 0 {
		payload.ID = makeID()
	}
	if len(payload.Pass) <= 0 {
		payload.Pass = randSeq(12)
	}
	payload.Token, _ = createJWTToken(payload.ID)
	payloadCpy := payload
	payload.Pass = bcryptPass(payload.Pass)

	p.mu.Lock()
	// check if it exists then update
	var exists bool
	for i := range p.cfg.Parties {
		if p.cfg.Parties[i].ID == payload.ID {
			p.cfg.Parties[i].Pass = payload.Pass
			p.cfg.Parties[i].Token = payload.Token
			exists = true
			break
		}
	}
	if !exists {
		p.cfg.Parties = append(p.cfg.Parties, payload)
	}
	p.mu.Unlock()

	err = p.updatefile()
	if err != nil {
		log.Printf("error saving passwords %v", err)
		http.Error(w, "can't update password list", http.StatusInternalServerError)
	}

	buf, _ := json.Marshal(payloadCpy)
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(buf)
}

func checkToken(r *http.Request) error {
	const bearerPrefix = "Bearer "
	if authhdr := r.Header.Get("Authorization"); len(authhdr) > 0 {
		if strings.HasPrefix(authhdr, bearerPrefix) {
			tkn := strings.TrimPrefix(authhdr, bearerPrefix)
			if tkn == Token {
				return nil
			}
			return errors.New("invalid token")
		}
		return errors.New("invalid token format")
	}
	return errors.New("missing Authorization header")
}

func makeID() string {
	b := make([]byte, 32)
	_, _ = cryptorand.Read(b)
	return hex.EncodeToString(b)
}

func bcryptPass(password string) string {
	str := fmt.Sprintf("vega%v", password)
	buf, _ := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	return string(buf)
}

var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
