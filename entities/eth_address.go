package entities

type EthereumAddress struct{ ID }

func NewEthereumAddress(id string) EthereumAddress {
	return EthereumAddress{ID: ID(id)}
}
