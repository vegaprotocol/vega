package main

import "github.com/jessevdk/go-flags"

type walletGenkey struct {
	RootPathOption
	PassphraseOption
	Name string `short:"n" long:"name" description:"Name of the wallet to user"`
}

func (opts *walletGenkey) Execute(_ []string) error {
	return nil
}

func Wallet(parser *flags.Parser) error {
	root, err := parser.AddCommand("wallet", "Create and manage wallets", "", &Empty{})
	if err != nil {
		return err
	}

	if _, err := root.AddCommand("genkey",
		"Generates a new keypar for a wallet",
		"Generate a new keypair for a wallet, this will implicitly generate a new wallet if none exist for the given name",
		&walletGenkey{
			RootPathOption: NewRootPathOption(),
		}); err != nil {
		return err
	}

	return nil
}
