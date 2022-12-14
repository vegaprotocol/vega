package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/printer"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/paths"
	coreversion "code.vegaprotocol.io/vega/version"
	"code.vegaprotocol.io/vega/wallet/api"
	walletnode "code.vegaprotocol.io/vega/wallet/api/node"
	networkStore "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"code.vegaprotocol.io/vega/wallet/version"
	"code.vegaprotocol.io/vega/wallet/wallets"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	signTransactionLong = cli.LongDesc(`
		Sign a transaction using the specified wallet and public key and bundle it as a
		raw transaction ready to be sent. The resulting transaction is base64-encoded and
		can be sent using the command "raw_transaction send".

		The transaction should be a Vega transaction formatted as a JSON payload, as follows:

		'{"commandName": {"someProperty": "someValue"} }'

		For vote submission, it will look like this:

		'{"voteSubmission": {"proposalId": "some-id", "value": "VALUE_YES"}}'

		Providing a network will allow the signed transaction to contain a valid 
		proof-of-work generated and attached automatically. If using in an offline
		environment then proof-of-work details should be supplied via the CLI options.
	`)

	signTransactionExample = cli.Examples(`
		# Sign a transaction offline with necessary information to generate a proof-of-work
		{{.Software}} transaction sign --wallet WALLET --pubkey PUBKEY --tx-height TX_HEIGHT --chain-id CHAIN_ID --tx-block-hash BLOCK_HASH --pow-difficulty POW_DIFF --pow-difficulty "sha3_24_rounds" TRANSACTION

		# Sign a transaction online generating proof-of-work automatically using the network to obtain the last block data
		{{.Software}} transaction sign --wallet WALLET --pubkey PUBKEY --network NETWORK TRANSACTION

		# To decode the result, save the result in a file and use the command
		# "base64"
		{{.Software}} transaction sign --wallet WALLET --pubkey PUBKEY --network NETWORK TRANSACTION > result.txt
		base64 --decode --input result.txt
	`)
)

type SignTransactionHandler func(api.AdminSignTransactionParams, *zap.Logger) (api.AdminSignTransactionResult, error)

func NewCmdSignTransaction(w io.Writer, rf *RootFlags) *cobra.Command {
	handler := func(params api.AdminSignTransactionParams, log *zap.Logger) (api.AdminSignTransactionResult, error) {
		vegaPaths := paths.New(rf.Home)

		ws, err := wallets.InitialiseStore(rf.Home)
		if err != nil {
			return api.AdminSignTransactionResult{}, fmt.Errorf("couldn't initialise wallets store: %w", err)
		}

		ns, err := networkStore.InitialiseStore(vegaPaths)
		if err != nil {
			return api.AdminSignTransactionResult{}, fmt.Errorf("couldn't initialise network store: %w", err)
		}

		signTx := api.NewAdminSignTransaction(ws, ns, func(hosts []string, retries uint64) (walletnode.Selector, error) {
			return walletnode.BuildRoundRobinSelectorWithRetryingNodes(log, hosts, retries)
		})

		rawResult, errDetails := signTx.Handle(context.Background(), params, jsonrpc.RequestMetadata{})
		if errDetails != nil {
			return api.AdminSignTransactionResult{}, errors.New(errDetails.Data)
		}
		return rawResult.(api.AdminSignTransactionResult), nil
	}

	return BuildCmdSignTransaction(w, handler, rf)
}

func BuildCmdSignTransaction(w io.Writer, handler SignTransactionHandler, rf *RootFlags) *cobra.Command {
	f := &SignTransactionFlags{}

	cmd := &cobra.Command{
		Use:     "sign",
		Short:   "Sign a transaction for offline use",
		Long:    signTransactionLong,
		Example: signTransactionExample,
		RunE: func(_ *cobra.Command, args []string) error {
			if aLen := len(args); aLen == 0 {
				return flags.ArgMustBeSpecifiedError("transaction")
			} else if aLen > 1 {
				return flags.TooManyArgsError("transaction")
			}
			f.RawTransaction = args[0]

			req, err := f.Validate()
			if err != nil {
				return err
			}

			log, err := buildCmdLogger(rf.Output, "info")
			if err != nil {
				return fmt.Errorf("failed to build a logger: %w", err)
			}

			resp, err := handler(req, log)
			if err != nil {
				return err
			}

			switch rf.Output {
			case flags.InteractiveOutput:
				PrintSignTransactionResponse(w, resp, rf)
			case flags.JSONOutput:
				return printer.FprintJSON(w, resp)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"Wallet holding the public key",
	)
	cmd.Flags().StringVarP(&f.PubKey,
		"pubkey", "k",
		"",
		"Public key of the key pair to use for signing (hex-encoded)",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"Path to the file containing the wallet's passphrase",
	)
	cmd.Flags().Uint64Var(&f.TxBlockHeight,
		"tx-height",
		0,
		"It should be close to the current block height when the transaction is applied, with a threshold of ~ - 150 blocks, not required if --network is set",
	)
	cmd.Flags().StringVar(&f.ChainID,
		"chain-id",
		"",
		"The identifier of the chain on which the command will be sent to, not required if --network is set",
	)
	cmd.Flags().StringVar(&f.TxBlockHash,
		"tx-block-hash",
		"",
		"The block-hash corresponding to tx-height which will be used to generate proof-of-work (hex encoded)",
	)
	cmd.Flags().Uint32Var(&f.PowDifficulty,
		"pow-difficulty",
		0,
		"The proof-of-work difficulty level",
	)
	cmd.Flags().StringVar(&f.PowHashFunction,
		"pow-hash-function",
		"",
		"The proof-of-work hash function to use to compute the proof-of-work",
	)
	cmd.Flags().StringVar(&f.Network,
		"network",
		"",
		"The network the transaction will be sent to",
	)

	autoCompleteWallet(cmd, rf.Home, "wallet")

	return cmd
}

type SignTransactionFlags struct {
	Wallet          string
	PubKey          string
	PassphraseFile  string
	RawTransaction  string
	TxBlockHeight   uint64
	ChainID         string
	TxBlockHash     string
	PowDifficulty   uint32
	PowHashFunction string
	Network         string
}

func (f *SignTransactionFlags) Validate() (api.AdminSignTransactionParams, error) {
	params := api.AdminSignTransactionParams{}

	if len(f.Wallet) == 0 {
		return api.AdminSignTransactionParams{}, flags.MustBeSpecifiedError("wallet")
	}
	params.Wallet = f.Wallet

	if len(f.PubKey) == 0 {
		return api.AdminSignTransactionParams{}, flags.MustBeSpecifiedError("pubkey")
	}
	if len(f.RawTransaction) == 0 {
		return api.AdminSignTransactionParams{}, flags.ArgMustBeSpecifiedError("transaction")
	}

	if f.Network == "" {
		if f.TxBlockHeight == 0 {
			return api.AdminSignTransactionParams{}, flags.MustBeSpecifiedError("tx-height")
		}

		if f.TxBlockHash == "" {
			return api.AdminSignTransactionParams{}, flags.MustBeSpecifiedError("tx-block-hash")
		}

		if f.ChainID == "" {
			return api.AdminSignTransactionParams{}, flags.MustBeSpecifiedError("chain-id")
		}
		if f.PowDifficulty == 0 {
			return api.AdminSignTransactionParams{}, flags.MustBeSpecifiedError("pow-difficulty")
		}
		if f.PowHashFunction == "" {
			return api.AdminSignTransactionParams{}, flags.MustBeSpecifiedError("pow-hash-function")
		}
		// populate proof-of-work bits
		params.LastBlockData = &api.AdminLastBlockData{
			ChainID:                 f.ChainID,
			BlockHeight:             f.TxBlockHeight,
			BlockHash:               f.TxBlockHash,
			ProofOfWorkDifficulty:   f.PowDifficulty,
			ProofOfWorkHashFunction: f.PowHashFunction,
		}
	}

	if f.Network != "" {
		if f.TxBlockHeight != 0 {
			return api.AdminSignTransactionParams{}, flags.MutuallyExclusiveError("network", "tx-height")
		}
		if f.TxBlockHash != "" {
			return api.AdminSignTransactionParams{}, flags.MutuallyExclusiveError("network", "tx-block-hash")
		}
		if f.ChainID != "" {
			return api.AdminSignTransactionParams{}, flags.MutuallyExclusiveError("network", "chain-id")
		}
		if f.PowDifficulty != 0 {
			return api.AdminSignTransactionParams{}, flags.MutuallyExclusiveError("network", "pow-difficulty")
		}
		if f.PowHashFunction != "" {
			return api.AdminSignTransactionParams{}, flags.MutuallyExclusiveError("network", "pow-hash-function")
		}
	}

	passphrase, err := flags.GetPassphrase(f.PassphraseFile)
	if err != nil {
		return api.AdminSignTransactionParams{}, err
	}
	params.Passphrase = passphrase

	params.Network = f.Network
	params.PublicKey = f.PubKey

	// Encode transaction into nested structure; this is a bit nasty but mirroring what happens
	// when our json-rpc library parses a request. There's an issue (6983#) to make the use
	// json.RawMessage instead.
	transaction := make(map[string]any)
	if err := json.Unmarshal([]byte(f.RawTransaction), &transaction); err != nil {
		return api.AdminSignTransactionParams{}, err
	}

	params.Transaction = transaction
	return params, nil
}

func PrintSignTransactionResponse(w io.Writer, req api.AdminSignTransactionResult, rf *RootFlags) {
	p := printer.NewInteractivePrinter(w)

	if rf.Output == flags.InteractiveOutput && version.IsUnreleased() {
		str := p.String()
		str.CrossMark().DangerText("You are running an unreleased version of the Vega wallet (").DangerText(coreversion.Get()).DangerText(").").NextLine()
		str.Pad().DangerText("Use it at your own risk!").NextSection()
		p.Print(str)
	}

	str := p.String()
	defer p.Print(str)
	str.CheckMark().SuccessText("Transaction signature successful").NextSection()
	str.Text("Transaction (base64-encoded):").NextLine().WarningText(req.EncodedTransaction).NextSection()

	str.BlueArrow().InfoText("Send a transaction").NextLine()
	str.Text("To send a raw transaction, see the following transaction:").NextSection()
	str.Code(fmt.Sprintf("%s raw_transaction send --help", os.Args[0])).NextSection()
}
