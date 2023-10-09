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

package paths

import (
	"context"

	"github.com/jessevdk/go-flags"
)

type Cmd struct {
	List    ListCmd    `command:"list"    description:"List the location where files used by the Vega applications are stored"`
	Explain ExplainCmd `command:"explain" description:"Explain what a path is about"`
}

var pathsCmd Cmd

func Paths(ctx context.Context, parser *flags.Parser) error {
	pathsCmd = Cmd{
		List:    ListCmd{},
		Explain: ExplainCmd{},
	}

	var (
		short = "Manages the Vega paths"
		long  = `
			Vega applications store their configuration and their data at specific locations.
			By default, it uses the XDG Base Directory specification, but can be customised
			using the --home flag.
			
			The XDG Base Directory specification defines where these files should be looked
			for by defining several base directories relative to which files should be 
			located. The location of these directories is specific to each platform.
			
			Role   | Linux          | MacOS                         | Windows
			-------| ---------------|-------------------------------|---------------------
			cache  | ~/.cache       | ~/Library/Caches              | %LOCALAPPDATA%\cache
			config | ~/.config      | ~/Library/Application Support | %LOCALAPPDATA%
			data   | ~/.local/share | ~/Library/Application Support | %LOCALAPPDATA% 
			state  | ~/.local/state | ~/Library/Application Support | %LOCALAPPDATA% 
			
			Vega applications also support setting a custom location for these files using
			the --home flag. Contrary to the XDG Base Directory specification, this flag
			will group the cache, config, data and state folders under a "vega" folder, at
			the specified location.`
	)

	_, err := parser.AddCommand("paths", short, long, &pathsCmd)

	return err
}
