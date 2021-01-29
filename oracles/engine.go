package oracles

import "errors"

// OracleData holds normalized data coming from an oracle.
type OracleData struct {
	Data    map[string]string
	PubKeys []string
}

// Engine is responsible of broadcasting the OracleData to products and risk
// models interested in it.
type Engine struct{}

// NewEngine creates a new oracle Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// BroadcastData broadcasts the OracleData to products and risk models that are interested in
// it. If no one is listening to this OracleData, it is discarded.
func (e *Engine) BroadcastData(data OracleData) error {
	return errors.New("not implemented")
}
