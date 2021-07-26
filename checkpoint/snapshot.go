package checkpoint

type Snapshot struct {
	name string
	hash []byte
	data []byte
}

func (s Snapshot) Name() string {
	return s.name
}

func (s Snapshot) Hash() []byte {
	return s.hash
}

func (s Snapshot) Data() []byte {
	return s.data
}
