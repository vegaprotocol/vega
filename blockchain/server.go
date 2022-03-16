package blockchain

type ChainServerImpl interface {
	ReloadConf(cfg Config)
	Stop()
	Start() error
}

// Server abstraction for the abci server.
type Server struct {
	*Config
	srv ChainServerImpl
}

// NewServer instantiate a new blockchain server.
func NewServer(srv ChainServerImpl) *Server {
	return &Server{
		srv: srv,
	}
}

func (s *Server) Start() error {
	return s.srv.Start()
}

// Stop gracefully shutdowns down the blockchain provider's server.
func (s *Server) Stop() {
	s.srv.Stop()
}

func (s *Server) ReloadConf(cfg Config) {
	s.srv.ReloadConf(cfg)
}
