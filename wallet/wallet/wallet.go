package wallet

// nolint: interfacebloat
type Wallet interface {
	KeyDerivationVersion() uint32
	Name() string
	SetName(newName string)
	ID() string
	Type() string
	HasPublicKey(pubKey string) bool
	DescribePublicKey(pubKey string) (PublicKey, error)
	DescribeKeyPair(pubKey string) (KeyPair, error)
	ListPublicKeys() []PublicKey
	ListKeyPairs() []KeyPair
	MasterKey() (MasterKeyPair, error)
	GenerateKeyPair(meta []Metadata) (KeyPair, error)
	TaintKey(pubKey string) error
	UntaintKey(pubKey string) error
	AnnotateKey(pubKey string, meta []Metadata) ([]Metadata, error)
	SignAny(pubKey string, data []byte) ([]byte, error)
	VerifyAny(pubKey string, data, sig []byte) (bool, error)
	SignTx(pubKey string, data []byte) (*Signature, error)
	IsIsolated() bool
	IsolateWithKey(pubKey string) (Wallet, error)
	Permissions(hostname string) Permissions
	PermittedHostnames() []string
	RevokePermissions(hostname string)
	PurgePermissions()
	UpdatePermissions(hostname string, perms Permissions) error
	Clone() Wallet
}

// nolint: interfacebloat
type KeyPair interface {
	PublicKey() string
	PrivateKey() string
	Name() string
	IsTainted() bool
	Metadata() []Metadata
	UpdateMetadata([]Metadata) []Metadata
	Index() uint32
	AlgorithmVersion() uint32
	AlgorithmName() string
	SignAny(data []byte) ([]byte, error)
	VerifyAny(data, sig []byte) (bool, error)
	Sign(data []byte) (*Signature, error)
}

type PublicKey interface {
	Key() string
	Name() string
	IsTainted() bool
	Metadata() []Metadata
	Index() uint32
	AlgorithmVersion() uint32
	AlgorithmName() string
	Hash() (string, error)
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

type MasterKeyPair interface {
	PublicKey() string
	PrivateKey() string
	AlgorithmVersion() uint32
	AlgorithmName() string
	SignAny(data []byte) ([]byte, error)
	Sign(data []byte) (*Signature, error)
}
