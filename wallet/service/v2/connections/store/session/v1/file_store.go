package v1

import (
	"context"
	"fmt"
	"sort"
	"sync"

	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
)

type FileStore struct {
	sessionsFilePath string

	mu sync.Mutex
}

func (s *FileStore) ListSessions(_ context.Context) ([]connections.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionsFile, err := s.readSessionsFile()
	if err != nil {
		return nil, err
	}

	sessions := make([]connections.Session, 0, len(sessionsFile.Sessions))

	for rawToken, session := range sessionsFile.Sessions {
		token, err := connections.AsToken(rawToken)
		if err != nil {
			return nil, fmt.Errorf("token %q is not a valid token: %w", rawToken, err)
		}
		sessions = append(sessions, connections.Session{
			Token:    token,
			Hostname: session.Hostname,
			Wallet:   session.Wallet,
		})
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].Hostname == sessions[j].Hostname {
			return sessions[i].Wallet < sessions[j].Wallet
		}

		return sessions[i].Hostname < sessions[j].Hostname
	})

	return sessions, nil
}

func (s *FileStore) TrackSession(session connections.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokensFile, err := s.readSessionsFile()
	if err != nil {
		return err
	}

	tokensFile.Sessions[session.Token.String()] = sessionContent{
		Hostname: session.Hostname,
		Wallet:   session.Wallet,
	}

	return s.writeSessionsFile(tokensFile)
}

func (s *FileStore) DeleteSession(_ context.Context, token connections.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionsFile, err := s.readSessionsFile()
	if err != nil {
		return err
	}

	delete(sessionsFile.Sessions, token.String())

	return s.writeSessionsFile(sessionsFile)
}

func (s *FileStore) readSessionsFile() (sessions sessionsFile, rerr error) {
	defer func() {
		if r := recover(); r != nil {
			sessions, rerr = sessionsFile{}, fmt.Errorf("a system error occurred while reading the tokens file: %s", r)
		}
	}()

	exists, err := vgfs.FileExists(s.sessionsFilePath)
	if err != nil {
		return sessionsFile{}, fmt.Errorf("could not verify the existence of the tokens file: %w", err)
	} else if !exists {
		return defaultSessionsFileContent(), nil
	}

	if err := paths.ReadStructuredFile(s.sessionsFilePath, &sessions); err != nil {
		return sessionsFile{}, fmt.Errorf("couldn't read the sessions file %s: %w", s.sessionsFilePath, err)
	}

	if sessions.FileVersion != 1 {
		return sessionsFile{}, fmt.Errorf("the sessions file is using the file format v%d but that format is not supported by this application", sessions.FileVersion)
	}

	if sessions.TokensVersion != 1 {
		return sessionsFile{}, fmt.Errorf("the tokens used in sessions are using the token format v%d but that format is not supported by this application", sessions.TokensVersion)
	}

	if sessions.Sessions == nil {
		sessions.Sessions = map[string]sessionContent{}
	}

	return sessions, nil
}

func (s *FileStore) writeSessionsFile(sessions sessionsFile) (rerr error) {
	defer func() {
		if r := recover(); r != nil {
			rerr = fmt.Errorf("a system error occurred while writing the sessions file:: %s", r)
		}
	}()
	if err := paths.WriteStructuredFile(s.sessionsFilePath, sessions); err != nil {
		return fmt.Errorf("couldn't write the sessions file %s: %w", s.sessionsFilePath, err)
	}

	return nil
}

func InitialiseStore(p paths.Paths) (*FileStore, error) {
	sessionsFilePath, err := p.CreateDataPathFor(paths.WalletServiceSessionTokensDataFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get data path for %s: %w", paths.WalletServiceSessionTokensDataFile, err)
	}

	store := &FileStore{
		sessionsFilePath: sessionsFilePath,
	}

	exists, err := vgfs.FileExists(sessionsFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not verify wether the session file exists or not: %w", err)
	}
	if !exists {
		err := paths.WriteStructuredFile(sessionsFilePath, defaultSessionsFileContent())
		if err != nil {
			return nil, fmt.Errorf("could not initialise the sessions file: %w", err)
		}
	}

	if _, err := store.readSessionsFile(); err != nil {
		return nil, err
	}

	return store, nil
}

type sessionsFile struct {
	FileVersion   int                       `json:"fileVersion"`
	TokensVersion int                       `json:"tokensVersion"`
	Sessions      map[string]sessionContent `json:"sessions"`
}

type sessionContent struct {
	Hostname string `json:"hostname"`
	Wallet   string `json:"wallet"`
}

func defaultSessionsFileContent() sessionsFile {
	return sessionsFile{
		FileVersion:   1,
		TokensVersion: 1,
		Sessions:      map[string]sessionContent{},
	}
}
