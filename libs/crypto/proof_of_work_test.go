package crypto_test

import (
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/libs/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoW(t *testing.T) {
	t.Parallel()

	_, _, err := crypto.PoW(crypto.RandomHash(), crypto.RandomHash(), 5, "nonExisting")
	require.Error(t, err)

	_, _, err = crypto.PoW(crypto.RandomHash(), crypto.RandomHash(), 257, "nonExisting")
	require.Error(t, err)

	blockHash := "2FB2146FC01F21D358323174BAA230E7DE61C0F150B7FBC415C896B0C23E50FF"
	txID := "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4"

	nonce, _, err := crypto.PoW(blockHash, txID, 2, crypto.Sha3)
	require.NoError(t, err)
	require.Equal(t, uint64(4), nonce)
	success, _ := crypto.Verify(blockHash, txID, nonce, crypto.Sha3, 2)
	require.True(t, success)
}

func TestVerify(t *testing.T) {
	t.Parallel()

	success, _ := crypto.Verify("", "", 0, "non existing", 0)
	require.False(t, false, success)
	success, _ = crypto.Verify("", "", 0, "non existing", 1)
	require.False(t, false, success)
	success, _ = crypto.Verify("", "", 0, crypto.Sha3, 1)
	require.False(t, false, success)
	success, _ = crypto.Verify("", "", 4, crypto.Sha3, 1)
	require.False(t, false, success)
	success, _ = crypto.Verify("", "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", 4, crypto.Sha3, 2)
	require.False(t, false, success)
	success, _ = crypto.Verify("2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", 4, crypto.Sha3, 3)
	require.False(t, false, success)
	success, _ = crypto.Verify("2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", 4, crypto.Sha3, 2)
	require.True(t, true, success)
	success, _ = crypto.Verify("2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", 4, crypto.Sha3, 1)
	require.True(t, true, success)
	success, _ = crypto.Verify("2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4", 4, crypto.Sha3, 0)
	require.True(t, true, success)
}

func TestCountZeros(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name       string
		blockHash  string
		txID       string
		difficulty uint
	}{
		{
			// 00001315c698aae3e559e9de507c43260e4b89e992840c281c68d54663eb02ae
			name:       "with difficulty set to 19",
			blockHash:  "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4",
			txID:       "DFE522E234D67E6AE3F017859F898E576B3928EA57310B765398615A0D3FDE2F",
			difficulty: 19,
		}, {
			// 00000d9ae20dd3c9ed57260ffe67832a98ccb43f797bba82f8a21be137e0ae5b
			name:       "with difficulty set to 20",
			blockHash:  "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4",
			txID:       "5B87F9DFA41DABE84A11CA78D9FE11DA8FC2AA926004CA66454A7AF0A206480D",
			difficulty: 20,
		}, {
			// 000003bbf0cde49e3899ad23282b18defbc12a65f07c95d768464b87024df368
			name:       "with difficulty set to 21",
			blockHash:  "2E7A16D9EF690F0D2BEED115FBA13BA2AAA16C8F971910AD88C72B9DB010C7D4",
			txID:       "B14DD602ED48C9F7B5367105A4A97FFC9199EA0C9E1490B786534768DD1538EF",
			difficulty: 21,
		}, {
			// 0000039c42f8c0a62ad39e1459393803104d8fdc2cd15410daaec3d8de7b85a0
			name:       "with difficulty set to 22",
			blockHash:  "B14DD602ED48C9F7B5367105A4A97FFC9199EA0C9E1490B786534768DD1538EF",
			txID:       "94A9CB1532011081B013CCD8E6AAA832CAB1CBA603F0C5A093B14C4961E5E7F0",
			difficulty: 22,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			_, hash, err := crypto.PoW(tc.blockHash, tc.txID, tc.difficulty, crypto.Sha3)
			require.NoError(tt, err)
			assert.NotEmpty(tt, hash)

			zeros := crypto.CountZeros(hash)

			require.Equal(tt, byte(tc.difficulty), zeros)
		})
	}
}

func TestDifficulty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		difficulty uint
		nonce      uint64
		blockHash  string
		tid        string
		proof      []byte
	}{
		{
			name:       "difficulty 4",
			difficulty: 4,
			nonce:      0,
			blockHash:  "792ca202b84226c739f9923046a0f4e7b5ff9e6f1b5636d8e26a8e2c5dec70ac",
			tid:        "3b8399cdffee2686d75d1a96d22cd49cd11f62c93da20e72239895bfdaf4b772",
			proof:      []byte("03f9f7d9911d3ca37c3356f10cd04273e788d1f57a9bc2396e7b5aa2e8d74557"),
		},
		{
			name:       "difficulty 8",
			difficulty: 8,
			nonce:      402,
			blockHash:  "ffb67ea4111d466d363a5c8f355bf81e2e3504563af273f5de81a005a6247e14",
			tid:        "c40de04280ce8c40ee41b5005c23a1358b4fbf31f6dcb675e8246b174458274e",
			proof:      []byte("0053ea7687bd7652803af4300a7e17868267c32e4fb7f09375c46c367fd7646b"),
		},
		{
			name:       "difficulty 12",
			difficulty: 12,
			nonce:      2560,
			blockHash:  "d9ae00ce4c4fc96d8e72bb18f6990b833cc7724ad70322604c572f6e194d777f",
			tid:        "fcbbb4cc8dcd402a07af050bb809a04bd82f9c95b6e5a56768d3724a4abb09f0",
			proof:      []byte("0008bbe071959bfe7fc426c4f378fcdb9540b3f931f4a0b09469f5bf0fddcb86"),
		},
		{
			name:       "difficulty 16",
			difficulty: 16,
			nonce:      23845,
			blockHash:  "dc4b61de2138856406acdabcc502be708bff7c945857ea032011a8b4b0cf54f4",
			tid:        "3954a15b2e1ec457ae100c56e2aa43786b4612644926403d59fd8cdcb29d825f",
			proof:      []byte("00000fd8f55699845ac3192af013928916050eab088437943708b83b27490862"),
		},
		{
			name:       "difficulty 20",
			difficulty: 20,
			nonce:      85863,
			blockHash:  "8890702af457ddcda01fba579a126adcecae954781500acb546fef9c8087a239",
			tid:        "74030ee7dc931be9d9cc5f2c9d44ac174b4144b377ef07a7bb1781856921dd43",
			proof:      []byte("000007542dcb39d1471fd6c7424a547b9039382e055ceed10c839f2b76f88c0d"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()

			n, h, err := crypto.PoW(tc.blockHash, tc.tid, tc.difficulty, crypto.Sha3)
			require.NoError(tt, err)
			require.Equal(tt, tc.nonce, n)
			require.Equal(tt, string(tc.proof), hex.EncodeToString(h))

			b, d := crypto.Verify(tc.blockHash, tc.tid, tc.nonce, crypto.Sha3, tc.difficulty)
			require.Equal(tt, true, b)
			require.True(tt, d >= byte(tc.difficulty))
		})
	}
}
