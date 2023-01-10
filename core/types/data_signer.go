package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

// PubKey.
type PubKey struct {
	Key string
}

func (p PubKey) IntoProto() *datapb.PubKey {
	pk := &datapb.PubKey{}
	if p.Key != "" {
		pk.Key = p.Key
	}
	return pk
}

func (p PubKey) String() string {
	return fmt.Sprintf(
		"pubKey(%s)",
		p.Key,
	)
}

func (p PubKey) DeepClone() *PubKey {
	if p.Key == "" {
		return &PubKey{}
	}

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
		return &SignerPubKey{}
	}
	return &SignerPubKey{
		PubKey: s.PubKey,
	}
}

func (s SignerPubKey) IsEmpty() bool {
	return s.PubKey.Key == ""
}

func (s SignerPubKey) AsHex(prepend bool) dataSourceSpec {
	// Check if the content is already hex encoded
	if strings.HasPrefix(s.PubKey.Key, "0x") {
		return &s
	}

	validHex, _ := isHex(s.PubKey.Key)
	if validHex {
		if prepend {
			s.PubKey.Key = fmt.Sprintf("0x%s", s.PubKey.Key)
		}
		return &s
	}

	// If the content is not a valid Hex - encode it
	s.PubKey.Key = fmt.Sprintf("0x%s", hex.EncodeToString([]byte(s.PubKey.Key)))
	return &s
}

func (s SignerPubKey) AsString() (dataSourceSpec, error) {
	// Check if the content is hex encoded
	st := strings.TrimPrefix(s.PubKey.Key, "0x")
	_, err := isHex(st)
	if err != nil {
		return &s, fmt.Errorf("signer is not a valid hex: %v", err)
	}

	decoded, err := hex.DecodeString(s.PubKey.Key)
	if err != nil {
		return &s, fmt.Errorf("error decoding signer: %v", err)
	}

	s.PubKey.Key = string(decoded)
	return &s, nil
}

func (s SignerPubKey) Serialize() []byte {
	c := strings.TrimPrefix(s.PubKey.Key, "0x")
	return append([]byte{byte(SignerPubKeyPrepend)}, []byte(c)...)
}

func DeserializePubKey(data []byte) SignerPubKey {
	return SignerPubKey{
		PubKey: &PubKey{
			Key: string(data),
		},
	}
}

func PubKeyFromProto(s *datapb.Signer_PubKey) *SignerPubKey {
	var pubKey *PubKey
	if s.PubKey != nil {
		pubKey = &PubKey{}

		if s.PubKey.Key != "" {
			pubKey.Key = s.PubKey.Key
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
	ethAddress := &datapb.ETHAddress{}
	if e.Address != "" {
		ethAddress.Address = e.Address
	}

	return ethAddress
}

func (e ETHAddress) String() string {
	return fmt.Sprintf(
		"ethAddress(%s)",
		e.Address,
	)
}

func (e ETHAddress) DeepClone() *ETHAddress {
	if e.Address == "" {
		return &ETHAddress{}
	}

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
		"signerETHAddres(%s)",
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
		return &SignerETHAddress{}
	}
	return &SignerETHAddress{
		ETHAddress: s.ETHAddress,
	}
}

func (s SignerETHAddress) IsEmpty() bool {
	return s.ETHAddress.Address == ""
}

func (s SignerETHAddress) AsHex(prepend bool) dataSourceSpec {
	// Check if the content is already hex encoded
	if strings.HasPrefix(s.ETHAddress.Address, "0x") {
		return &s
	}

	validHex, _ := isHex(s.ETHAddress.Address)
	if validHex {
		if prepend {
			s.ETHAddress.Address = fmt.Sprintf("0x%s", s.ETHAddress.Address)
		}
		return &s
	}

	s.ETHAddress.Address = fmt.Sprintf("0x%s", hex.EncodeToString([]byte(s.ETHAddress.Address)))
	return &s
}

func (s SignerETHAddress) AsString() (dataSourceSpec, error) {
	// Check if the content is hex encoded
	st := strings.TrimPrefix(s.ETHAddress.Address, "0x")
	_, err := isHex(st)
	if err != nil {
		return &s, fmt.Errorf("signer is not a valid hex: %v", err)
	}

	decoded, err := hex.DecodeString(s.ETHAddress.Address)
	if err != nil {
		return &s, fmt.Errorf("error decoding signer: %v", err)
	}

	s.ETHAddress.Address = string(decoded)
	return &s, nil
}

func (s SignerETHAddress) Serialize() []byte {
	c := strings.TrimPrefix(s.ETHAddress.Address, "0x")
	return append([]byte{byte(ETHAddressPrepend)}, []byte(c)...)
}

func DeserializeETHAddress(data []byte) SignerETHAddress {
	return SignerETHAddress{
		ETHAddress: &ETHAddress{
			Address: "0x" + string(data),
		},
	}
}

func ETHAddressFromProto(s *datapb.Signer_EthAddress) *SignerETHAddress {
	var ethAddress *ETHAddress
	if s.EthAddress != nil {
		ethAddress = &ETHAddress{}

		if s.EthAddress.Address != "" {
			ethAddress.Address = s.EthAddress.Address
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
	AsHex(bool) dataSourceSpec
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

func (s Signer) IntoProto() *datapb.Signer {
	sig := s.Signer.oneOfProto()
	signer := &datapb.Signer{}

	switch tp := sig.(type) {
	case *datapb.Signer_PubKey:
		signer.Signer = tp
	case *datapb.Signer_EthAddress:
		signer.Signer = tp
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
	return s.Signer.IsEmpty()
}

func (s Signer) GetSignerPubKey() *PubKey {
	switch t := s.Signer.(type) {
	case *SignerPubKey:
		return t.PubKey
	default:
		return nil
	}
}

func (s Signer) GetSignerETHAddress() *ETHAddress {
	switch t := s.Signer.(type) {
	case *SignerETHAddress:
		return t.ETHAddress
	default:
		return nil
	}
}

func (s Signer) GetSignerType() DataSignerType {
	switch s.Signer.(type) {
	case *SignerPubKey:
		return DataSignerTypePubKey
	case *SignerETHAddress:
		return DataSignerTypeEthAddress
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
			sign := signer.oneOfProto()
			protoSigners[i] = sign.(*datapb.Signer)
			// protoSigners[i] = signer.IntoProto()
		}
	}

	return protoSigners
}

func SignersToStringList(s []*Signer) []string {
	var signers []string

	if len(s) > 0 {
		for _, signer := range s {
			signers = append(signers, signer.String())
		}
		return signers
	}
	return signers
}

// SignersFromProto returns a list of signers after checking the list length.
func SignersFromProto(s []*datapb.Signer) []*Signer {
	signers := []*Signer{}
	if len(s) > 0 {
		signers = make([]*Signer, len(s))
		for i, signer := range s {
			signers[i] = SignerFromProto(signer)
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
func SignerAsHex(signer *Signer) *Signer {
	switch signer.GetSignerType() {
	case DataSignerTypePubKey:
		s := signer.Signer.(*SignerPubKey).AsHex(false)
		return &Signer{s}

	case DataSignerTypeEthAddress:
		s := signer.Signer.(*SignerETHAddress).AsHex(false)
		return &Signer{s}
	}

	// If the signer type is not among the ones we know, we do not care to
	// encode or decode it.
	return nil
}

// SignerAsString represents the Signer content as a string.
func SignerAsString(signer *Signer) (*Signer, error) {
	switch signer.GetSignerType() {
	case DataSignerTypePubKey:
		s, err := signer.Signer.(*SignerPubKey).AsString()
		return &Signer{s}, err
	case DataSignerTypeEthAddress:
		s, err := signer.Signer.(*SignerETHAddress).AsString()
		return &Signer{s}, err
	}

	// If the signer type is not among the ones we know, we do not care to
	// encode or decode it.
	return nil, fmt.Errorf("unknown signer type")
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
		return s.Signer.(*SignerPubKey).Serialize(), nil

	case DataSignerTypeEthAddress:
		return s.Signer.(*SignerETHAddress).Serialize(), nil
	}

	// If the signer type is not among the ones we know, we do not care to
	// encode or decode it.
	return nil, fmt.Errorf("error from serializing signer: unknown signer type")
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

func NewSigner() *Signer {
	return &Signer{}
}
