package admin

import (
	"fmt"
	"net/http"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/nodewallets/registry"
)

type wallet interface {
	Name() string
	PubKey() crypto.PublicKey
}

type Wallet struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

func (w *Wallet) String() string {
	return fmt.Sprintf("Name: %s, PublicKey: %s", w.Name, w.PublicKey)
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
	log                  *logging.Logger
	nodeWallets          *nodewallets.NodeWallets
	registryLoader       *registry.Loader
	nodeWalletPassphrase string
}

func NewNodeWallet(
	log *logging.Logger,
	vegaPaths paths.Paths,
	nodeWalletPassphrase string,
	nodeWallets *nodewallets.NodeWallets,
) (*NodeWallet, error) {
	registryLoader, err := registry.NewLoader(vegaPaths, nodeWalletPassphrase)
	if err != nil {
		return nil, err
	}

	return &NodeWallet{
		log:                  log,
		nodeWallets:          nodeWallets,
		registryLoader:       registryLoader,
		nodeWalletPassphrase: nodeWalletPassphrase,
	}, nil
}

func (h *NodeWallet) Reload(r *http.Request, args *NodeWalletArgs, reply *NodeWalletReloadReply) error {
	h.log.Info("Reloading node wallet", logging.String("chain", args.Chain))

	switch args.Chain {
	case "vega":
		oW := newWallet(h.nodeWallets.Vega)

		reg, err := h.registryLoader.Get(h.nodeWalletPassphrase)
		if err != nil {
			return fmt.Errorf("couldn't load node wallet registry: %v", err)
		}

		if err := h.nodeWallets.Vega.Reload(*reg.Vega); err != nil {
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

		reg, err := h.registryLoader.Get(h.nodeWalletPassphrase)
		if err != nil {
			return fmt.Errorf("couldn't load node wallet registry: %v", err)
		}

		if err := h.nodeWallets.Ethereum.Reload(reg.Ethereum.Details); err != nil {
			h.log.Error("Reloading node wallet failed", logging.Error(err))
			return fmt.Errorf("failed to reload Ethereum wallet: %w", err)
		}

		nW := newWallet(h.nodeWallets.Ethereum)

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
