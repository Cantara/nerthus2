package dir

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
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
		inf, err := d.Info()
		if err != nil {
			return err
		}
		mode := strconv.FormatUint(uint64(inf.Mode().Perm()), 8)
		modType := strconv.FormatUint(uint64(inf.Mode().Type()>>27), 8)
		mode = modType + mode[:len(mode)-1] + "0"
		files = append(files, file.File{
			Name:    nodeDir + strings.TrimPrefix(path, filesDir),
			Mode:    mode,
			Content: string(b),
		})
		return nil
	})
	return
}
