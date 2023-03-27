package dirReader

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/cantara/nerthus2/configManager/file"
)

func ReadFilesFromDir(envFS fs.FS, configDir, localDir, nodeDir string) (files []file.File, err error) {
	filesDir := filepath.Clean(fmt.Sprintf("%s/files/%s", configDir, localDir))
	err = fs.WalkDir(envFS, filesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(envFS, path)
		if err != nil {
			return err
		}
		files = append(files, file.File{
			Name:    nodeDir + strings.TrimPrefix(path, filesDir),
			Content: string(b),
		})
		return nil
	})
	return
}
