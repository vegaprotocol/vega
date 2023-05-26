package netparams

import "context"

type patchDesc struct {
	Key      string // the key to update
	Value    string
	Validate bool
	// can be nil, but can be used to set the value based on an older value
	SetValue func(ctx context.Context, p *patchDesc, s *Store) error
}
