package types

type EthCallSpecConfiguration struct {
	ID        string
	Time      int64
	Address   string
	ABI       []byte
	Method    string
	Arguments []string
	Signers   []*Signer
}

type oracleSourceType interface {
	isOracleSourceType()
	oneOfProto() interface{}
	String() string
	DeepClone() oracleSourceType
}

func (e *EthCallSpecConfiguration) isOracleSourceType() {}

func (e *EthCallSpecConfiguration) oneOfProto() interface{} {
	return e.IntoProto()
}

func (e *EthCallSpecConfiguration) IntoProto() *vegapb.EthCallSpecConfiguration {

}

func (e *EthCallSpecConfiguration) String() string {

}

func (e *EthCallSpecConfiguration) DeepClone() oracleSourceType {

}

func EthCallSpecConfigurationFromProto() *EthCallSpecConfiguration {

}

type EthTimeTrigger struct {
	Initial int64
	Every int64
	Until int64
}

