// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package paths

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/core/config"
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
