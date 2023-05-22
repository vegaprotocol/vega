package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

var (
	ErrSignerIsEmpty     = errors.New("signer is empty")
	ErrSignerInValidHex  = errors.New("signer is not a valid hex")
	ErrSignerUnknownType = errors.New("unknown type of signer")
)

// PubKey.
type PubKey struct {
	Key string
}

func (p PubKey) IntoProto() *datapb.PubKey {
	return &datapb.PubKey{
		Key: p.Key,
	}
}

func (p PubKey) String() string {
	return fmt.Sprintf(
		"pubKey(%s)",
		p.Key,
	)
}

func (p PubKey) DeepClone() *PubKey {
	return &PubKey{
		Key: p.Key,
	}
}

type SignerPubKey struct {
	PubKey *PubKey
}

func (s SignerPubKey) isDataSourceSpec() {}

func (s SignerPubKey) String() string {
	return fmt.Sprintf(
		"signerPubKey(%s)",
		reflectPointerToString(s.PubKey),
	)
}

func (s SignerPubKey) IntoProto() *datapb.Signer_PubKey {
	pubKey := &datapb.PubKey{}
	if s.PubKey != nil {
		pubKey = s.PubKey.IntoProto()
	}

	return &datapb.Signer_PubKey{
		PubKey: pubKey,
	}
}

func (s SignerPubKey) oneOfProto() interface{} {
	return s.IntoProto()
}

func (s SignerPubKey) GetSignerType() DataSignerType {
	return DataSignerTypePubKey
}

func (s SignerPubKey) DeepClone() dataSourceSpec {
	if s.PubKey == nil {
		return &SignerPubKey{} // ?
	}
	return &SignerPubKey{
		PubKey: s.PubKey,
	}
}

func (s SignerPubKey) IsEmpty() bool {
	if s.PubKey == nil {
		return true
	}
	return s.PubKey.Key == ""
}

func (s SignerPubKey) AsHex(prepend bool) (dataSourceSpec, error) {
	if s.PubKey == nil {
		return &s, ErrSignerIsEmpty
	}

	if s.PubKey.Key == "" {
		return nil, ErrSignerIsEmpty
	}

	// Check if the content is already hex encoded
	if strings.HasPrefix(s.PubKey.Key, "0x") {
		return &s, nil
	}

	validHex, _ := isHex(s.PubKey.Key)
	if validHex {
		if prepend {
			s.PubKey.Key = fmt.Sprintf("0x%s", s.PubKey.Key)
		}
		return &s, nil
	}

	// If the content is not a valid Hex - encode it
	s.PubKey.Key = fmt.Sprintf("0x%s", hex.EncodeToString([]byte(s.PubKey.Key)))
	return &s, nil
}

func (s SignerPubKey) AsString() (dataSourceSpec, error) {
	if s.PubKey == nil {
		return nil, ErrSignerIsEmpty
	}

	// Check if the content is hex encoded
	st := strings.TrimPrefix(s.PubKey.Key, "0x")
	validHex, _ := isHex(st)
	if validHex {
		decoded, err := hex.DecodeString(st)
		if err != nil {
			return &s, fmt.Errorf("error decoding signer: %v", err)
		}

		s.PubKey.Key = string(decoded)
	}
	return &s, nil
}

func (s SignerPubKey) Serialize() []byte {
	c := strings.TrimPrefix(s.PubKey.Key, "0x")
	return append([]byte{byte(SignerPubKeyPrepend)}, []byte(c)...)
}

func DeserializePubKey(data []byte) *SignerPubKey {
	return &SignerPubKey{
		PubKey: &PubKey{
			Key: string(data),
		},
	}
}

func PubKeyFromProto(s *datapb.Signer_PubKey) *SignerPubKey {
	var pubKey *PubKey
	if s != nil {
		if s.PubKey != nil {
			pubKey = &PubKey{
				Key: s.PubKey.Key,
			}
		}
	}

	return &SignerPubKey{
		PubKey: pubKey,
	}
}

// ETHAddress.
type ETHAddress struct {
	Address string
}

func (e ETHAddress) IntoProto() *datapb.ETHAddress {
	return &datapb.ETHAddress{
		Address: e.Address,
	}
}

func (e ETHAddress) String() string {
	return fmt.Sprintf(
		"ethAddress(%s)",
		e.Address,
	)
}

func (e ETHAddress) DeepClone() *ETHAddress {
	return &ETHAddress{
		Address: e.Address,
	}
}

type SignerETHAddress struct {
	ETHAddress *ETHAddress
}

func (s SignerETHAddress) isDataSourceSpec() {}

func (s SignerETHAddress) String() string {
	return fmt.Sprintf(
		"signerETHAddress(%s)",
		reflectPointerToString(s.ETHAddress),
	)
}

func (s SignerETHAddress) IntoProto() *datapb.Signer_EthAddress {
	ethAddress := &datapb.ETHAddress{}
	if s.ETHAddress != nil {
		ethAddress = s.ETHAddress.IntoProto()
	}

	return &datapb.Signer_EthAddress{
		EthAddress: ethAddress,
	}
}

func (s SignerETHAddress) oneOfProto() interface{} {
	return s.IntoProto()
}

func (s SignerETHAddress) GetSignerType() DataSignerType {
	return DataSignerTypeEthAddress
}

func (s SignerETHAddress) DeepClone() dataSourceSpec {
	if s.ETHAddress == nil {
		return &SignerETHAddress{} // ?
	}
	return &SignerETHAddress{
		ETHAddress: s.ETHAddress,
	}
}

func (s SignerETHAddress) IsEmpty() bool {
	if s.ETHAddress == nil {
		return true
	}
	return s.ETHAddress.Address == ""
}

func (s SignerETHAddress) AsHex(prepend bool) (dataSourceSpec, error) {
	if s.ETHAddress == nil {
		return nil, ErrSignerIsEmpty
	}

	if s.ETHAddress.Address == "" {
		return nil, ErrSignerIsEmpty
	}

	// Check if the content is already hex encoded
	if strings.HasPrefix(s.ETHAddress.Address, "0x") {
		return &s, nil
	}

	validHex, _ := isHex(s.ETHAddress.Address)
	if validHex {
		if prepend {
			s.ETHAddress.Address = fmt.Sprintf("0x%s", s.ETHAddress.Address)
		}
		return &s, nil
	}

	s.ETHAddress.Address = fmt.Sprintf("0x%s", hex.EncodeToString([]byte(s.ETHAddress.Address)))
	return &s, nil
}

func (s SignerETHAddress) AsString() (dataSourceSpec, error) {
	if s.ETHAddress == nil {
		return nil, ErrSignerIsEmpty
	}

	// Check if the content is hex encoded
	st := strings.TrimPrefix(s.ETHAddress.Address, "0x")
	validHex, _ := isHex(st)
	if validHex {
		decoded, err := hex.DecodeString(st)
		if err != nil {
			return &s, fmt.Errorf("error decoding signer: %v", err)
		}

		s.ETHAddress.Address = string(decoded)
	}
	return &s, nil
}

func (s SignerETHAddress) Serialize() []byte {
	c := strings.TrimPrefix(s.ETHAddress.Address, "0x")
	return append([]byte{byte(ETHAddressPrepend)}, []byte(c)...)
}

func DeserializeETHAddress(data []byte) *SignerETHAddress {
	return &SignerETHAddress{
		ETHAddress: &ETHAddress{
			Address: "0x" + string(data),
		},
	}
}

func ETHAddressFromProto(s *datapb.Signer_EthAddress) *SignerETHAddress {
	var ethAddress *ETHAddress
	if s != nil {
		if s.EthAddress != nil {
			ethAddress = &ETHAddress{
				Address: s.EthAddress.Address,
			}
		}
	}

	return &SignerETHAddress{
		ETHAddress: ethAddress,
	}
}

type DataSignerType int

const (
	DataSignerTypeUnspecified DataSignerType = iota
	DataSignerTypePubKey
	DataSignerTypeEthAddress
)

type dataSourceSpec interface {
	isDataSourceSpec()
	oneOfProto() interface{}
	DeepClone() dataSourceSpec
	GetSignerType() DataSignerType
	AsHex(bool) (dataSourceSpec, error)
	AsString() (dataSourceSpec, error)
	String() string
	IsEmpty() bool
}

type Signer struct {
	Signer dataSourceSpec
}

// func (s Signer) isDataSourceSpec() {}

func (s Signer) oneOfProto() interface{} {
	return s.IntoProto()
}

// IntoProto will always return a `datapb.Signer` that needs to be checked
// if it has any content afterwards.
func (s Signer) IntoProto() *datapb.Signer {
	signer := &datapb.Signer{}
	if s.Signer != nil {
		sig := s.Signer.oneOfProto()

		switch tp := sig.(type) {
		case *datapb.Signer_PubKey:
			signer.Signer = tp
		case *datapb.Signer_EthAddress:
			signer.Signer = tp
		}
	}

	return signer
}

func (s Signer) DeepClone() *Signer {
	cpy := s
	cpy.Signer = s.Signer.DeepClone()
	return &cpy
}

func (s Signer) String() string {
	return reflectPointerToString(s.Signer)
}

func (s Signer) IsEmpty() bool {
	if s.Signer != nil {
		return s.Signer.IsEmpty()
	}
	return true
}

func (s Signer) GetSignerPubKey() *PubKey {
	if s.Signer != nil {
		switch t := s.Signer.(type) {
		case *SignerPubKey:
			return t.PubKey
		}
	}
	return nil
}

func (s Signer) GetSignerETHAddress() *ETHAddress {
	if s.Signer != nil {
		switch t := s.Signer.(type) {
		case *SignerETHAddress:
			return t.ETHAddress
		}
	}
	return nil
}

func (s Signer) GetSignerType() DataSignerType {
	if s.Signer != nil {
		switch s.Signer.(type) {
		case *SignerPubKey:
			return DataSignerTypePubKey
		case *SignerETHAddress:
			return DataSignerTypeEthAddress
		}
	}

	return DataSignerTypeUnspecified
}

func SignerFromProto(s *datapb.Signer) *Signer {
	signer := &Signer{}

	if s.Signer != nil {
		switch t := s.Signer.(type) {
		case *datapb.Signer_PubKey:
			signer.Signer = PubKeyFromProto(t)
		case *datapb.Signer_EthAddress:
			signer.Signer = ETHAddressFromProto(t)
		}
	}

	return signer
}

func CreateSignerFromString(s string, t DataSignerType) *Signer {
	signer := &Signer{}
	switch t {
	case DataSignerTypePubKey:
		signer.Signer = &SignerPubKey{PubKey: &PubKey{s}}
	case DataSignerTypeEthAddress:
		signer.Signer = &SignerETHAddress{ETHAddress: &ETHAddress{s}}
	}

	return signer
}

// SignersIntoProto returns a list of signers after checking the list length.
func SignersIntoProto(s []*Signer) []*datapb.Signer {
	protoSigners := []*datapb.Signer{}
	if len(s) > 0 {
		protoSigners = make([]*datapb.Signer, len(s))
		for i, signer := range s {
			if signer != nil {
				sign := signer.oneOfProto()
				protoSigners[i] = sign.(*datapb.Signer)
			}
		}
	}

	return protoSigners
}

func SignersToStringList(s []*Signer) []string {
	var signers []string

	if len(s) > 0 {
		for _, signer := range s {
			if signer != nil {
				signers = append(signers, signer.String())
			}
		}
		return signers
	}
	return signers
}

// SignersFromProto returns a list of signers.
// The list is allowed to be empty.
func SignersFromProto(s []*datapb.Signer) []*Signer {
	signers := []*Signer{}
	if len(s) > 0 {
		signers = make([]*Signer, len(s))
		for i, signer := range s {
			if s != nil {
				signers[i] = SignerFromProto(signer)
			}
		}
		return signers
	}
	return signers
}

// Encoding and decoding options

const (
	SignerPubKeyPrepend = 0x00
	ETHAddressPrepend   = 0x01
)

// SignerAsHex represents the signer as a hex encoded string.
// We export this function as a standalone option because there are cases when we are not sure
// what is the signer type we deal with.
func SignerAsHex(signer *Signer) (*Signer, error) {
	switch signer.GetSignerType() {
	case DataSignerTypePubKey:
		if signer.Signer != nil {
			s, err := signer.Signer.(*SignerPubKey).AsHex(false)
			return &Signer{s}, err
		}
		return nil, ErrSignerIsEmpty

	case DataSignerTypeEthAddress:
		if signer.Signer != nil {
			s, err := signer.Signer.(*SignerETHAddress).AsHex(false)
			return &Signer{s}, err
		}
		return nil, ErrSignerIsEmpty
	}

	// If the signer type is not among the ones we know, we do not care to
	// encode or decode it.
	return nil, ErrSignerUnknownType
}

// SignerAsString represents the Signer content as a string.
func SignerAsString(signer *Signer) (*Signer, error) {
	switch signer.GetSignerType() {
	case DataSignerTypePubKey:
		if signer.Signer != nil {
			s, err := signer.Signer.(*SignerPubKey).AsString()

			return &Signer{s}, err
		}
		return nil, ErrSignerIsEmpty

	case DataSignerTypeEthAddress:
		if signer.Signer != nil {
			s, err := signer.Signer.(*SignerETHAddress).AsString()
			return &Signer{s}, err
		}
		return nil, ErrSignerIsEmpty
	}

	// If the signer type is not among the ones we know, we do not care to
	// encode or decode it.
	return nil, ErrSignerUnknownType
}

func isHex(src string) (bool, error) {
	dst := make([]byte, hex.DecodedLen(len(src)))
	if _, err := hex.Decode(dst, []byte(src)); err != nil {
		return false, fmt.Errorf("string is not a valid hex: %v", err)
	}

	return true, nil
}

// Serialization and deserialization

// SerializeSigner deserializes the signer to a byte slice - we use that
// type top insert it into the database.
// The deserialization prepends the slice with two bytes as signer type indicator -
// that is used when the Signer is serialized back.
func (s *Signer) Serialize() ([]byte, error) {
	switch s.GetSignerType() {
	case DataSignerTypePubKey:
		if s.Signer != nil {
			pk, _ := s.Signer.(*SignerPubKey)
			if pk.PubKey != nil {
				return pk.Serialize(), nil
			}
		}
		return nil, ErrSignerIsEmpty

	case DataSignerTypeEthAddress:
		if s.Signer != nil {
			ea, _ := s.Signer.(*SignerETHAddress)
			if ea.ETHAddress != nil {
				return ea.Serialize(), nil
			}
		}
		return nil, ErrSignerIsEmpty
	}

	// If the signer type is not among the ones we know, we do not care to
	// encode or decode it.
	return nil, ErrSignerUnknownType
}

func DeserializeSigner(content []byte) *Signer {
	if len(content) > 0 {
		switch content[0] {
		case SignerPubKeyPrepend:
			return &Signer{
				Signer: DeserializePubKey(content[1:]),
			}

		case ETHAddressPrepend:
			return &Signer{
				Signer: DeserializeETHAddress(content[1:]),
			}
		}
	}
	// If the signer type is not among the ones we know, we do not care to
	// encode or decode it.
	return &Signer{Signer: nil}
}

func NewSigner(t DataSignerType) *Signer {
	switch t {
	case DataSignerTypePubKey:
		return &Signer{
			Signer: &SignerPubKey{},
		}

	case DataSignerTypeEthAddress:
		return &Signer{
			Signer: &SignerETHAddress{},
		}
	}
	// No indication if the given type is unknown.
	return nil
}
