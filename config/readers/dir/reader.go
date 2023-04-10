package dir

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/cantara/nerthus2/config/readers/file"
)

func ReadFilesFromDir(sysFS fs.FS, localDir, nodeDir string) (files []file.File, err error) {
	filesDir := filepath.Clean(fmt.Sprintf("files/%s", localDir))
	err = fs.WalkDir(sysFS, filesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(sysFS, path)
		if err != nil {
			return err
		}
		files = append(files, file.File{
			Name:    nodeDir + strings.TrimPrefix(path, filesDir),
			Content: b,
		})
		return nil
	})
	return
}
