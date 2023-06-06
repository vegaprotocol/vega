package types_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
)

func TestSignerIsEmpty(t *testing.T) {
	t.Run("empty singer", func(t *testing.T) {
		s := &types.Signer{}
		isEmpty := s.IsEmpty()
		assert.True(t, isEmpty)

		s = &types.Signer{
			Signer: &types.SignerETHAddress{},
		}
		isEmpty = s.IsEmpty()
		assert.True(t, isEmpty)

		s = &types.Signer{
			Signer: &types.SignerETHAddress{
				ETHAddress: nil,
			},
		}
		isEmpty = s.IsEmpty()
		assert.True(t, isEmpty)

		s = &types.Signer{
			Signer: &types.SignerETHAddress{
				ETHAddress: &types.ETHAddress{
					Address: "",
				},
			},
		}
		s = &types.Signer{
			Signer: &types.SignerETHAddress{
				ETHAddress: nil,
			},
		}
		isEmpty = s.IsEmpty()
		assert.True(t, isEmpty)

		s = &types.Signer{
			Signer: &types.SignerPubKey{
				PubKey: nil,
			},
		}
		isEmpty = s.IsEmpty()
		assert.True(t, isEmpty)

		s = &types.Signer{
			Signer: &types.SignerPubKey{},
		}
		isEmpty = s.IsEmpty()
		assert.True(t, isEmpty)

		s = &types.Signer{
			Signer: &types.SignerPubKey{
				PubKey: &types.PubKey{
					Key: "",
				},
			},
		}
		isEmpty = s.IsEmpty()
		assert.True(t, isEmpty)
	})
}

func TestCreateSignerFromString(t *testing.T) {
	signerString := "TESTSTRING"
	signer := types.CreateSignerFromString(signerString, types.DataSignerTypePubKey)
	assert.NotNil(t, signer)
	assert.NotNil(t, signer.Signer)
	// Implicitly test `GetSignerPubKey`
	assert.IsType(t, &types.PubKey{}, signer.GetSignerPubKey())
	assert.Equal(t, "TESTSTRING", signer.GetSignerPubKey().Key)

	signerString = "0xTESTSTRING"
	signer = types.CreateSignerFromString(signerString, types.DataSignerTypeEthAddress)
	assert.NotNil(t, signer)
	assert.NotNil(t, signer.Signer)
	// Implicitly test `GetSignerETHAddress`
	assert.IsType(t, &types.ETHAddress{}, signer.GetSignerETHAddress())
	assert.Equal(t, "0xTESTSTRING", signer.GetSignerETHAddress().Address)
}

func TestSignersIntoProto(t *testing.T) {
	signers := []*types.Signer{
		{
			Signer: &types.SignerPubKey{
				PubKey: &types.PubKey{
					Key: "testsign",
				},
			},
		},
		{
			Signer: &types.SignerETHAddress{
				ETHAddress: &types.ETHAddress{
					Address: "0xtest-ethereum-address",
				},
			},
		},
		{
			Signer: &types.SignerETHAddress{
				ETHAddress: nil,
			},
		},
	}

	protoSigners := types.SignersIntoProto(signers)
	assert.Equal(t, 3, len(protoSigners))
	assert.NotNil(t, protoSigners[0].GetPubKey())
	assert.IsType(t, &datapb.Signer_PubKey{}, protoSigners[0].Signer)
	assert.IsType(t, &datapb.PubKey{}, protoSigners[0].GetPubKey())
	assert.Equal(t, "testsign", protoSigners[0].GetPubKey().Key)
	assert.NotNil(t, protoSigners[1].GetEthAddress())
	assert.IsType(t, &datapb.Signer_EthAddress{}, protoSigners[1].Signer)
	assert.IsType(t, &datapb.ETHAddress{}, protoSigners[1].GetEthAddress())
	assert.Equal(t, "0xtest-ethereum-address", protoSigners[1].GetEthAddress().Address)
	assert.NotNil(t, protoSigners[2].GetEthAddress())
	assert.IsType(t, &datapb.Signer_EthAddress{}, protoSigners[1].Signer)
	assert.IsType(t, &datapb.ETHAddress{}, protoSigners[0].GetEthAddress())
	assert.Equal(t, "", protoSigners[2].GetEthAddress().Address)
}

func TestSignersToStringList(t *testing.T) {
	signers := []*types.Signer{
		{
			Signer: &types.SignerPubKey{
				PubKey: &types.PubKey{
					Key: "testsign",
				},
			},
		},
		{
			Signer: &types.SignerETHAddress{
				ETHAddress: &types.ETHAddress{
					Address: "0xtest-ethereum-address",
				},
			},
		},
		{
			Signer: &types.SignerETHAddress{
				ETHAddress: nil,
			},
		},
	}

	list := types.SignersToStringList(signers)
	assert.Equal(
		t,
		[]string{
			"signerPubKey(pubKey(testsign))",
			"signerETHAddress(ethAddress(0xtest-ethereum-address))",
			"signerETHAddress(nil)",
		},
		list,
	)
}

func TestSignersFromProto(t *testing.T) {
	t.Run("empty signers list", func(t *testing.T) {
		protoSigners := []*datapb.Signer{
			{},
			{},
		}

		signers := types.SignersFromProto(protoSigners)
		assert.Equal(t, 2, len(signers))
		for _, s := range signers {
			assert.Nil(t, s.Signer)
			assert.Nil(t, s.GetSignerPubKey())
			assert.Nil(t, s.GetSignerETHAddress())
		}

		protoSigners = []*datapb.Signer{
			{
				Signer: &datapb.Signer_PubKey{
					PubKey: &datapb.PubKey{},
				},
			},
			{
				Signer: &datapb.Signer_EthAddress{
					EthAddress: &datapb.ETHAddress{},
				},
			},
		}

		signers = types.SignersFromProto(protoSigners)
		assert.Equal(t, 2, len(signers))
		for i, s := range signers {
			assert.NotNil(t, s.Signer)
			if i == 0 {
				assert.NotNil(t, s.GetSignerPubKey())
				assert.Equal(t, "", s.GetSignerPubKey().Key)
			} else {
				assert.NotNil(t, s.GetSignerETHAddress())
				assert.Equal(t, "", s.GetSignerETHAddress().Address)
			}
		}
	})

	t.Run("non-empty signers list", func(t *testing.T) {
		protoSigners := []*datapb.Signer{
			{
				Signer: &datapb.Signer_PubKey{
					PubKey: &datapb.PubKey{
						Key: "TESTSIGN",
					},
				},
			},
			{
				Signer: &datapb.Signer_EthAddress{
					EthAddress: &datapb.ETHAddress{
						Address: "0xtest-ethereum-address",
					},
				},
			},
			{
				Signer: &datapb.Signer_PubKey{
					PubKey: &datapb.PubKey{},
				},
			},
			{
				Signer: &datapb.Signer_EthAddress{
					EthAddress: &datapb.ETHAddress{},
				},
			},
		}

		signers := types.SignersFromProto(protoSigners)
		assert.Equal(t, 4, len(signers))
		for i, s := range signers {
			assert.NotNil(t, s.Signer)
			if i == 0 {
				assert.NotNil(t, s.GetSignerPubKey())
				assert.Equal(t, "TESTSIGN", s.GetSignerPubKey().Key)
			}
			if i == 1 {
				assert.NotNil(t, s.GetSignerETHAddress())
				assert.Equal(t, "0xtest-ethereum-address", s.GetSignerETHAddress().Address)
			}

			if i == 2 {
				assert.NotNil(t, s.GetSignerPubKey())
				assert.Equal(t, "", s.GetSignerPubKey().Key)
			}

			if i == 3 {
				assert.NotNil(t, s.GetSignerETHAddress())
				assert.Equal(t, "", s.GetSignerETHAddress().Address)
			}
		}
	})
}

func TestSignerAsHex(t *testing.T) {
	t.Run("empty signer", func(t *testing.T) {
		signer := &types.Signer{}

		hexSigner, err := types.SignerAsHex(signer)
		assert.ErrorIs(t, types.ErrSignerUnknownType, err)
		assert.Nil(t, hexSigner)

		signer = &types.Signer{Signer: &types.SignerPubKey{}}

		hexSigner, err = types.SignerAsHex(signer)
		assert.ErrorIs(t, types.ErrSignerIsEmpty, err)
		assert.NotNil(t, hexSigner)
		assert.NotNil(t, hexSigner.Signer)
		assert.Nil(t, hexSigner.GetSignerPubKey())

		signer = &types.Signer{
			Signer: &types.SignerETHAddress{
				ETHAddress: &types.ETHAddress{},
			},
		}

		hexSigner, err = types.SignerAsHex(signer)
		assert.ErrorIs(t, types.ErrSignerIsEmpty, err)
		assert.NotNil(t, hexSigner)
		assert.Nil(t, hexSigner.Signer)
		assert.Nil(t, hexSigner.GetSignerETHAddress())

		signer = &types.Signer{
			Signer: &types.SignerPubKey{
				PubKey: &types.PubKey{},
			},
		}

		hexSigner, err = types.SignerAsHex(signer)
		assert.ErrorIs(t, types.ErrSignerIsEmpty, err)
		assert.NotNil(t, hexSigner)
		assert.Nil(t, hexSigner.Signer)
		assert.Nil(t, hexSigner.GetSignerPubKey())
	})

	t.Run("non-empty pubKey signer", func(t *testing.T) {
		signer := &types.Signer{
			Signer: &types.SignerPubKey{
				PubKey: &types.PubKey{
					Key: "TESTSIGN",
				},
			},
		}

		hexSigner, err := types.SignerAsHex(signer)
		assert.Nil(t, err)
		assert.IsType(t, &types.Signer{}, hexSigner)
		assert.NotNil(t, hexSigner.Signer)
		assert.IsType(t, &types.SignerPubKey{}, hexSigner.Signer)
		assert.NotNil(t, hexSigner.Signer.GetSignerType())
		assert.IsType(t, &types.PubKey{}, hexSigner.GetSignerPubKey())
		assert.Equal(t, "0x544553545349474e", hexSigner.GetSignerPubKey().Key)
	})

	t.Run("non-empty ethAddress signer", func(t *testing.T) {
		signer := &types.Signer{
			Signer: &types.SignerETHAddress{
				ETHAddress: &types.ETHAddress{
					Address: "0xtest-ethereum-address",
				},
			},
		}

		hexSigner, err := types.SignerAsHex(signer)
		assert.Nil(t, err)
		assert.IsType(t, &types.Signer{}, hexSigner)
		assert.NotNil(t, hexSigner.Signer)
		assert.IsType(t, &types.SignerETHAddress{}, hexSigner.Signer)
		assert.NotNil(t, hexSigner.Signer.GetSignerType())
		assert.IsType(t, &types.ETHAddress{}, hexSigner.GetSignerETHAddress())
		assert.Equal(t, "0xtest-ethereum-address", hexSigner.GetSignerETHAddress().Address)
	})
}

func TestSignerAsString(t *testing.T) {
	t.Run("empty signer", func(t *testing.T) {
		signer := &types.Signer{}
		signAsString, err := types.SignerAsString(signer)
		assert.ErrorIs(t, types.ErrSignerUnknownType, err)
		assert.Nil(t, signAsString)
	})

	t.Run("empty pubkey/eth address signer", func(t *testing.T) {
		signer := &types.Signer{
			Signer: &types.SignerPubKey{
				PubKey: nil,
			},
		}

		signAsString, err := types.SignerAsString(signer)
		assert.ErrorIs(t, types.ErrSignerIsEmpty, err)
		assert.NotNil(t, signAsString)
		assert.IsType(t, &types.Signer{}, signAsString)
		assert.Nil(t, signAsString.Signer)

		signer = &types.Signer{
			Signer: &types.SignerETHAddress{
				ETHAddress: nil,
			},
		}

		signAsString, err = types.SignerAsString(signer)
		assert.ErrorIs(t, types.ErrSignerIsEmpty, err)
		assert.NotNil(t, signAsString)
		assert.IsType(t, &types.Signer{}, signAsString)
		assert.Nil(t, signAsString.Signer)
	})

	t.Run("non-empty pubkey signer", func(t *testing.T) {
		signer := &types.Signer{
			Signer: &types.SignerPubKey{
				PubKey: &types.PubKey{
					Key: "testsign",
				},
			},
		}

		signAsString, err := types.SignerAsString(signer)
		assert.ErrorIs(t, nil, err)
		assert.NotNil(t, signAsString)
		assert.IsType(t, &types.Signer{}, signAsString)
		assert.NotNil(t, signAsString.Signer)
		assert.NotNil(t, signAsString.GetSignerPubKey())
		assert.Equal(t, "testsign", signAsString.GetSignerPubKey().Key)

		signer = &types.Signer{
			Signer: &types.SignerPubKey{
				PubKey: &types.PubKey{
					Key: "0x544553545349474e",
				},
			},
		}

		signAsString, err = types.SignerAsString(signer)
		assert.ErrorIs(t, nil, err)
		assert.NotNil(t, signAsString)
		assert.IsType(t, &types.Signer{}, signAsString)
		assert.NotNil(t, signAsString.Signer)
		assert.NotNil(t, signAsString.GetSignerPubKey())
		assert.Equal(t, "TESTSIGN", signAsString.GetSignerPubKey().Key)
	})

	t.Run("non-empty eth address signer", func(t *testing.T) {
		signer := &types.Signer{
			Signer: &types.SignerETHAddress{
				ETHAddress: &types.ETHAddress{
					Address: "0x746573742d657468657265756d2d61646472657373",
				},
			},
		}

		signAsString, err := types.SignerAsString(signer)
		assert.ErrorIs(t, nil, err)
		assert.NotNil(t, signAsString)
		assert.IsType(t, &types.Signer{}, signAsString)
		assert.NotNil(t, signAsString.Signer)
		assert.NotNil(t, signAsString.GetSignerETHAddress())
		assert.Equal(t, "test-ethereum-address", signAsString.GetSignerETHAddress().Address)
	})
}

func TestSignerSerialize(t *testing.T) {
	t.Run("empty signer", func(t *testing.T) {
		signer := &types.Signer{
			Signer: nil,
		}

		serialized, err := signer.Serialize()
		assert.ErrorIs(t, types.ErrSignerUnknownType, err)
		assert.Nil(t, serialized)
	})

	t.Run("empty pubKey signer", func(t *testing.T) {
		signer := &types.Signer{
			Signer: &types.SignerPubKey{},
		}

		serialized, err := signer.Serialize()
		assert.ErrorIs(t, types.ErrSignerIsEmpty, err)
		assert.Nil(t, serialized)
	})

	t.Run("empty ethAddress signer", func(t *testing.T) {
		signer := &types.Signer{
			Signer: &types.SignerETHAddress{},
		}

		serialized, err := signer.Serialize()
		assert.ErrorIs(t, types.ErrSignerIsEmpty, err)
		assert.Nil(t, serialized)
	})

	t.Run("pubKey signer", func(t *testing.T) {
		key := "TESTKEY"
		signer := &types.Signer{
			Signer: &types.SignerPubKey{
				PubKey: &types.PubKey{
					Key: key,
				},
			},
		}

		// Test implicitly types.SignerPubKey.Serialize()
		serialized, err := signer.Serialize()
		assert.NoError(t, err)
		assert.Equal(t, uint8(0x0), serialized[0])
		assert.Equal(t, key, string(serialized[1:]))
	})

	t.Run("ethAddress signer", func(t *testing.T) {
		address := "test-eth-address"
		signer := &types.Signer{
			Signer: &types.SignerETHAddress{
				ETHAddress: &types.ETHAddress{
					Address: address,
				},
			},
		}

		// Tests implicitly types.SignerETHAddress.Serialize()
		serialized, err := signer.Serialize()
		assert.NoError(t, err)
		assert.Equal(t, uint8(0x1), serialized[0])
		assert.Equal(t, address, string(serialized[1:]))
	})
}

func TestDeserializeSigner(t *testing.T) {
	t.Run("empty content", func(t *testing.T) {
		signer := types.DeserializeSigner([]byte{})
		assert.NotNil(t, signer)
		assert.Nil(t, signer.Signer)
	})

	t.Run("non-empty content with no indicative byte", func(t *testing.T) {
		signer := types.DeserializeSigner([]byte{83, 84, 75, 69, 89})
		assert.NotNil(t, signer)
		assert.Nil(t, signer.Signer)
	})

	t.Run("non-empty pubKey with indicative byte", func(t *testing.T) {
		// Implicitly test DeserializePubKey
		signer := types.DeserializeSigner([]byte{0, 84, 69, 83, 84, 75, 69, 89})
		assert.NotNil(t, signer)
		assert.NotNil(t, signer.Signer)
		assert.IsType(t, &types.SignerPubKey{}, signer.Signer)
		assert.NotNil(t, signer.GetSignerPubKey())
		assert.Equal(t, "TESTKEY", signer.GetSignerPubKey().Key)
	})

	t.Run("non-empty ethAddress with indicative byte", func(t *testing.T) {
		// Implicitly test DeserializeETHAddress
		signer := types.DeserializeSigner([]byte{1, 116, 101, 115, 116, 45, 101, 116, 104, 45, 97, 100, 100, 114, 101, 115, 115})
		assert.NotNil(t, signer)
		assert.NotNil(t, signer.Signer)
		assert.IsType(t, &types.SignerETHAddress{}, signer.Signer)
		assert.NotNil(t, signer.GetSignerETHAddress())
		assert.Equal(t, "0xtest-eth-address", signer.GetSignerETHAddress().Address)
	})
}

func TestNewSigner(t *testing.T) {
	signer := types.NewSigner(types.DataSignerTypePubKey)
	assert.NotNil(t, signer)
	assert.NotNil(t, signer.Signer)
	assert.IsType(t, &types.SignerPubKey{}, signer.Signer)

	signer = types.NewSigner(types.DataSignerTypeEthAddress)
	assert.NotNil(t, signer)
	assert.NotNil(t, signer.Signer)
	assert.IsType(t, &types.SignerETHAddress{}, signer.Signer)

	signer = types.NewSigner(types.DataSignerType(0))
	assert.Nil(t, signer)
}
