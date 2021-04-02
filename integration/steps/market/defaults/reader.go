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
