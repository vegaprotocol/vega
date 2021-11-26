package blockchain

type ChainServerImpl interface {
	ReloadConf(cfg Config)
	Stop()
}

// Server abstraction for the abci server. Very small, and mostly so that we can just have a dummy one for if we have a nullchain
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

// Stop gracefully shutdowns down the blockchain provider's server
func (s *Server) Stop() {
	s.srv.Stop()
}

func (s *Server) ReloadConf(cfg Config) {
	s.srv.ReloadConf(cfg)
}
