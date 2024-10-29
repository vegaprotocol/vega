// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package emit_withdrawals

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/bridges"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/paths"
)

const (
	addressIndex  = 1
	assetIndex    = 2
	amountIndex   = 4
	creationIndex = 5
	nonceIndex    = 6
)

var (
	bundlesPath string
	out         string
	passphrase  config.Passphrase
	home        string
)

var (
	bridgeAddressMapping = map[string]string{
		"WETH":     "0x23872549cE10B40e31D6577e0A920088B0E0666a",
		"USDT ETH": "0x23872549cE10B40e31D6577e0A920088B0E0666a",
		"USDT ARB": "0x475B597652bCb2769949FD6787b1DC6916518407",
		"USDC":     "0x23872549cE10B40e31D6577e0A920088B0E0666a",
	}

	assetAddressMapping = map[string]string{
		"WETH":     "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
		"USDT ETH": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		"USDT ARB": "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",
		"USDC":     "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
	}

	bridgeChainIDMapping = map[string]string{
		"0x23872549cE10B40e31D6577e0A920088B0E0666a": "",
		"0x475B597652bCb2769949FD6787b1DC6916518407": "42161",
	}
)

func init() {
	flag.StringVar(&out, "out", "rebundled.csv", "where to store the outputs rebundled signatures")
	flag.StringVar(&home, "home", "", "path to the vega home root")
	flag.StringVar((*string)(&passphrase), "passphrase", "", "passphrase of the node wallet")
	flag.StringVar(&bundlesPath, "bundles", "", "path to the signatures bundles (required)")
}

func readCSV(path string) [][]string {
	buf, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("couldn't open the bundles file: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(buf))
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	return records[1:]
}

func signBundle(
	s bridges.Signer,
	token, amountStr, receiver, creationStr, nonceStr string,
) string {
	var (
		bridgeAddress = bridgeAddressMapping[token]
		tokenAddress  = assetAddressMapping[token]
		chainID       = bridgeChainIDMapping[bridgeAddress]
		erc20Logic    = bridges.NewERC20Logic(s, bridgeAddress, chainID, chainID == "")
	)

	nonce, overflowed := num.UintFromString(nonceStr, 10)
	if overflowed {
		log.Fatalf("invalid nonce, needs to be base 10, got: %v", nonceStr)
	}

	amount, overflowed := num.UintFromString(amountStr, 10)
	if overflowed {
		log.Fatalf("invalid amount, needs to be base 10, got: %v", amountStr)
	}

	creationInt64, err := strconv.ParseInt(creationStr, 10, 64)
	if err != nil {
		panic(err)
	}
	creation := time.Unix(creationInt64, 0)

	bundle, err := erc20Logic.WithdrawAsset(
		tokenAddress, amount, receiver, creation, nonce,
	)
	if err != nil {
		log.Fatalf("couldn't sign bundle: %v", err)
	}

	return fmt.Sprintf("0x%v", bundle.Signature.Hex())
}

func signAllBundles(s bridges.Signer, bundles [][]string) [][]string {
	entries := [][]string{}
	for _, entry := range bundles {
		signature := signBundle(s,
			entry[assetIndex],
			entry[amountIndex],
			entry[addressIndex],
			entry[creationIndex],
			entry[nonceIndex],
		)

		entries = append(entries, append(entry, signature))
	}

	return entries
}

func Main() {
	flag.Parse()

	if len(bundlesPath) <= 0 {
		log.Fatal("-bundles argument is required")
	}

	bundles := readCSV(bundlesPath)

	pass, err := passphrase.Get("node wallet", false)
	if err != nil {
		log.Fatalf("couldn't get the node wallet passphrase: %v", err)
	}

	vegaPaths := paths.New(home)
	if _, _, err := config.EnsureNodeConfig(vegaPaths); err != nil {
		log.Fatalf("couldn't load vega configuration: %v", err)
	}

	s, err := nodewallets.GetEthereumWallet(vegaPaths, pass)
	if err != nil {
		log.Fatalf("couldn't get Ethereum node wallet: %v", err)
	}

	output := signAllBundles(s, bundles)

	// just sort it to get the same output across nodes
	sort.Slice(output, func(i, j int) bool {
		return output[i][creationIndex] < output[j][creationIndex]
	})

	// then add headers
	output = append([][]string{
		{
			"party", "address", "asset", "realAmount", "amount", "createdTimestamp", "nonce", "signature",
		},
	}, output...)

	f, err := os.Create(out)
	if err != nil {
		log.Fatalf("couldn't create file: %v", err)
	}

	w := csv.NewWriter(f)
	w.WriteAll(output)

	if err := w.Error(); err != nil {
		log.Fatalf("error writing output:", err)
	}
}
