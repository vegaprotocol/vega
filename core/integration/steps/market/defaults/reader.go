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

package defaults

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

func ReadAll(embeddedModels embed.FS, modelNames []string) map[string]*bytes.Reader {
	contents := map[string]*bytes.Reader{}
	for _, modelName := range modelNames {
		content, err := fs.ReadFile(embeddedModels, modelName)
		if err != nil {
			panic(fmt.Errorf("failed to read file %s: %v", modelName, err))
		}
		stat, err := fs.Stat(embeddedModels, modelName)
		if err != nil {
			panic(fmt.Errorf("failed to get stat of file %s: %v", modelName, err))
		}
		fileNameWithoutExt := strings.TrimSuffix(stat.Name(), path.Ext(stat.Name()))
		contents[fileNameWithoutExt] = bytes.NewReader(content)
	}
	return contents
}
