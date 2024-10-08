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
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/paths"
)

type ListCmd struct {
	config.VegaHomeFlag
}

func (opts *ListCmd) Execute(_ []string) error {
	vegaPaths := paths.New(opts.VegaHome)

	allPaths := paths.List(vegaPaths)

	if err := printTable("Cache", allPaths.CachePaths); err != nil {
		return fmt.Errorf("couldn't print cache paths table: %w", err)
	}
	fmt.Print("\n\n")
	if err := printTable("Config", allPaths.ConfigPaths); err != nil {
		return fmt.Errorf("couldn't print config paths table: %w", err)
	}
	fmt.Print("\n\n")
	if err := printTable("Data", allPaths.DataPaths); err != nil {
		return fmt.Errorf("couldn't print data paths table: %w", err)
	}
	fmt.Print("\n\n")
	if err := printTable("State", allPaths.StatePaths); err != nil {
		return fmt.Errorf("couldn't print state paths table: %w", err)
	}

	return nil
}

func printTable(role string, paths map[string]string) error {
	sortedPaths := make([]struct {
		name string
		path string
	}, len(paths))

	var i uint64
	for name, path := range paths {
		sortedPaths[i] = struct {
			name string
			path string
		}{
			name: name,
			path: path,
		}
		i++
	}

	sort.SliceStable(sortedPaths, func(i, j int) bool {
		return sortedPaths[i].path < sortedPaths[j].path
	})

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)

	_, _ = fmt.Fprintf(w, "\n  %s\t%s\t", "NAME", "PATH")
	for _, path := range sortedPaths {
		_, _ = fmt.Fprintf(w, "\n  %s\t%s\t", path.name, path.path)
	}

	fmt.Printf("# %s paths\n\n", role)
	if err := w.Flush(); err != nil {
		return fmt.Errorf("couldn't flush paths table: %w", err)
	}

	return nil
}
