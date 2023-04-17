package dir

import (
	"bytes"
	"encoding/base64"
	"fmt"
	log "github.com/cantara/bragi/sbragi"
	"io"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cantara/nerthus2/config/readers/file"
	"github.com/gabriel-vasile/mimetype"
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
		mtype := mimetype.Detect(b)
		log.Debug("mime", "extension", mtype.Extension(), "string", mtype.String())
		isBinary := !strings.HasPrefix(mtype.String(), "text")
		content := string(b)
		if isBinary {
			buf := bytes.Buffer{}
			encoder := base64.NewEncoder(base64.StdEncoding, &buf)
			err = WriteAll(encoder, b)
			if err != nil {
				return err
			}
			content = buf.String()
		}
		files = append(files, file.File{
			Name:    nodeDir + strings.TrimPrefix(path, filesDir),
			Binary:  isBinary,
			Mode:    mode,
			Content: content,
		})
		return nil
	})
	return
}

func WriteAll(w io.Writer, data []byte) (err error) {
	totalOut := 0
	var n int
	for totalOut < len(data) {
		n, err = w.Write(data[totalOut:])
		if err != nil {
			log.WithError(err).Error("while writing all")
			return
		}
		totalOut += n
	}
	return
}
