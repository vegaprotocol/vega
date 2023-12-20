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

package nodewallet

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/admin"
	"code.vegaprotocol.io/vega/core/config"
	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type reloadCmd struct {
	config.OutputFlag

	Config admin.Config

	Chain string `choice:"vega" choice:"ethereum" description:"The chain to be imported" long:"chain" required:"true" short:"c"`
}

func (opts *reloadCmd) Execute(_ []string) error {
	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	vegaPaths := paths.New(rootCmd.VegaHome)

	_, conf, err := config.EnsureNodeConfig(vegaPaths)
	if err != nil {
		return err
	}

	opts.Config = conf.Admin

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	sc := admin.NewClient(opts.Config)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var resp *admin.NodeWalletReloadReply
	switch opts.Chain {
	case vegaChain, ethereumChain:
		resp, err = sc.NodeWalletReload(ctx, opts.Chain)
		if err != nil {
			return fmt.Errorf("failed to reload node wallet: %w", err)
		}
	default:
		return fmt.Errorf("chain %q is not supported", opts.Chain)
	}
	if output.IsHuman() {
		fmt.Println(green("reload successful:"))

		vgjson.PrettyPrint(resp)
	} else if output.IsJSON() {
		if err := vgjson.Print(resp); err != nil {
			return err
		}
	}

	return nil
}
