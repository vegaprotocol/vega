package core

type State struct {
	Size    int64  `json:"size"`
	Height  int64  `json:"height"`
	AppHash []byte `json:"app_hash"`
}

func newState() *State {
	return &State{}
}
