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

package helpers

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
