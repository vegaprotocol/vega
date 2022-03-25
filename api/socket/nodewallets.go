package socket

import (
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
)

type wallet interface {
	Name() string
	PubKey() crypto.PublicKey
}

type Wallet struct {
	Name      string
	PublicKey string
}

func newWallet(w wallet) Wallet {
	return Wallet{
		Name:      w.Name(),
		PublicKey: w.PubKey().Hex(),
	}
}

type NodeWalletArgs struct {
	Chain string
}

type NodeWalletReloadReply struct {
	OldWallet Wallet
	NewWallet Wallet
}

type NodeWallet struct {
	log         *logging.Logger
	nodeWallets *nodewallets.NodeWallets
}

func newNodeWallet(log *logging.Logger, nodeWallets *nodewallets.NodeWallets) *NodeWallet {
	return &NodeWallet{
		log:         log,
		nodeWallets: nodeWallets,
	}
}

func (h *NodeWallet) Reload(r *http.Request, args *NodeWalletArgs, reply *NodeWalletReloadReply) error {
	h.log.Info("Reloading node wallet", logging.String("chain", args.Chain))

	switch args.Chain {
	case "vega":
		oW := newWallet(h.nodeWallets.Vega)

		if err := h.nodeWallets.ReloadVega(); err != nil {
			h.log.Error("Reloading node wallet failed", logging.Error(err))
			return fmt.Errorf("failed to reload Vega wallet: %w", err)
		}

		nW := newWallet(h.nodeWallets.Vega)

		reply.NewWallet = nW
		reply.OldWallet = oW

		h.log.Info("Reloaded node wallet", logging.String("chain", args.Chain))
		return nil
	case "ethereum":
		oW := newWallet(h.nodeWallets.Ethereum)

		fmt.Println("---- ow:", oW)
		if err := h.nodeWallets.ReloadEthereum(); err != nil {
			h.log.Error("Reloading node wallet failed", logging.Error(err))
			return fmt.Errorf("failed to reload Ethereum wallet: %w", err)
		}

		nW := newWallet(h.nodeWallets.Ethereum)

		fmt.Println("---- ow:", nW)

		reply.NewWallet = nW
		reply.OldWallet = oW

		h.log.Info("Reloaded node wallet", logging.String("chain", args.Chain))
		return nil
	}

	return fmt.Errorf("failed to reload wallet for non existing chain %q", args.Chain)
}

func (h *NodeWallet) Show(r *http.Request, args *NodeWalletArgs, reply *Wallet) error {
	switch args.Chain {
	case "vega":
		*reply = newWallet(h.nodeWallets.Vega)
		return nil
	case "ethereum":
		*reply = newWallet(h.nodeWallets.Ethereum)
		return nil
	}

	return fmt.Errorf("failed to show wallet for non existing chain %q", args.Chain)
}
