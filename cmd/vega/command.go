package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/go-wallet/wallet"
	wstore "code.vegaprotocol.io/go-wallet/wallet/store/v1"
	"code.vegaprotocol.io/protos/commands"
	"code.vegaprotocol.io/protos/vega/api"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/config"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jessevdk/go-flags"
)

type CommandCmd struct {
	config.RootPathFlag

	NodeAddress string `long:"node-address" description:"The address of the vega node to use" default:"0.0.0.0:3002"`

	Passphrase config.Passphrase `short:"p" long:"passphrase" description:"A file containing the passphrase for the wallet, if empty will prompt for input"`

	WalletName string `long:"wallet-name" required:"true" description:"Name of the wallet to use to send the transaction"`

	Pubkey string `long:"pubkey" required:"true" description:"The wallet pubkey to be used"`
}

var commandCmd CommandCmd

func Command(ctx context.Context, parser *flags.Parser) error {
	rootPath := config.NewRootPathFlag()
	nodeCmd = NodeCmd{
		RootPathFlag: rootPath,
	}
	_, err := parser.AddCommand("command", "Send a command to a vega network", "Send a command to a vega network", &commandCmd)
	if err != nil {
		return err
	}

	return nil
}

func (opts *CommandCmd) Execute(args []string) error {
	if len(args) <= 0 {
		return errors.New("missing command payload")
	}

	command := walletpb.SubmitTransactionRequest{}
	err := jsonpb.UnmarshalString(args[0], &command)
	if err != nil {
		return fmt.Errorf("invalid command input: %w", err)
	}

	command.PubKey = opts.Pubkey

	err = wallet.CheckSubmitTransactionRequest(&command)
	if err != nil {
		return fmt.Errorf("invalid command payload: %w", err)
	}

	store, err := wstore.NewStore(filepath.Join(opts.RootPath, "wallets"))
	if err != nil {
		return fmt.Errorf("could not load wallet: %w", err)
	}

	passphrase, err := opts.Passphrase.Get("wallet")
	if err != nil {
		return fmt.Errorf("could not get the wallet passphrase: %w", err)
	}

	w, err := store.GetWallet(opts.WalletName, passphrase)
	if err != nil {
		return fmt.Errorf("could not open wallet: %w", err)
	}

	clt, err := getClient(opts.NodeAddress)
	if err != nil {
		return fmt.Errorf("could not connect to vega node: %w", err)
	}

	height, err := getHeight(clt)
	if err != nil {
		return err
	}

	tx, err := signTx(w, &command, height)
	if err != nil {
		return fmt.Errorf("could not sign the transaction: %w", err)
	}

	return sendTx(clt, tx)
}

func sendTx(clt api.TradingServiceClient, tx *commandspb.Transaction) error {
	ctx, cfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cfunc()
	req := api.SubmitTransactionV2Request{
		Tx:   tx,
		Type: api.SubmitTransactionV2Request_TYPE_ASYNC,
	}
	_, err := clt.SubmitTransactionV2(ctx, &req)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}
	return nil
}

func getHeight(clt api.TradingServiceClient) (uint64, error) {
	ctx, cfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cfunc()
	resp, err := clt.LastBlockHeight(ctx, &api.LastBlockHeightRequest{})
	if err != nil {
		return 0, fmt.Errorf("could not get last block: %w", err)
	}

	return resp.Height, nil
}

func signTx(w wallet.Wallet, req *walletpb.SubmitTransactionRequest, height uint64) (*commandspb.Transaction, error) {
	data := commands.NewInputData(height)
	wrapRequestCommandIntoInputData(data, req)
	marshalledData, err := proto.Marshal(data)
	if err != nil {
		return nil, err
	}

	pubKey := req.GetPubKey()
	signature, err := w.SignTxV2(pubKey, marshalledData)
	if err != nil {
		return nil, err
	}

	return commands.NewTransaction(pubKey, marshalledData, signature), nil
}

func wrapRequestCommandIntoInputData(data *commandspb.InputData, req *walletpb.SubmitTransactionRequest) error {
	switch cmd := req.Command.(type) {
	case *walletpb.SubmitTransactionRequest_OrderSubmission:
		data.Command = &commandspb.InputData_OrderSubmission{
			OrderSubmission: req.GetOrderSubmission(),
		}
	case *walletpb.SubmitTransactionRequest_OrderCancellation:
		data.Command = &commandspb.InputData_OrderCancellation{
			OrderCancellation: req.GetOrderCancellation(),
		}
	case *walletpb.SubmitTransactionRequest_OrderAmendment:
		data.Command = &commandspb.InputData_OrderAmendment{
			OrderAmendment: req.GetOrderAmendment(),
		}
	case *walletpb.SubmitTransactionRequest_VoteSubmission:
		data.Command = &commandspb.InputData_VoteSubmission{
			VoteSubmission: req.GetVoteSubmission(),
		}
	case *walletpb.SubmitTransactionRequest_WithdrawSubmission:
		data.Command = &commandspb.InputData_WithdrawSubmission{
			WithdrawSubmission: req.GetWithdrawSubmission(),
		}
	case *walletpb.SubmitTransactionRequest_LiquidityProvisionSubmission:
		data.Command = &commandspb.InputData_LiquidityProvisionSubmission{
			LiquidityProvisionSubmission: req.GetLiquidityProvisionSubmission(),
		}
	case *walletpb.SubmitTransactionRequest_ProposalSubmission:
		data.Command = &commandspb.InputData_ProposalSubmission{
			ProposalSubmission: req.GetProposalSubmission(),
		}
	case *walletpb.SubmitTransactionRequest_ChainEvent:
		data.Command = &commandspb.InputData_ChainEvent{
			ChainEvent: req.GetChainEvent(),
		}
	case *walletpb.SubmitTransactionRequest_OracleDataSubmission:
		data.Command = &commandspb.InputData_OracleDataSubmission{
			OracleDataSubmission: req.GetOracleDataSubmission(),
		}
	case *walletpb.SubmitTransactionRequest_DelegateSubmission:
		data.Command = &commandspb.InputData_DelegateSubmission{
			DelegateSubmission: req.GetDelegateSubmission(),
		}
	case *walletpb.SubmitTransactionRequest_UndelegateSubmission:
		data.Command = &commandspb.InputData_UndelegateSubmission{
			UndelegateSubmission: req.GetUndelegateSubmission(),
		}
	default:
		return fmt.Errorf("command %v is not supported", cmd)
	}
	return nil
}

func getClient(address string) (api.TradingServiceClient, error) {
	tdconn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return api.NewTradingServiceClient(tdconn), nil
}
