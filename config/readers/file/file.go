package file

import (
	"path/filepath"
	"strings"
)

type File struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content"`
}

func FilesFromConfig(fileMap map[string]string) (files []File) {
	files = make([]File, len(fileMap))
	fileNum := 0
	for fn, content := range fileMap {
		files[fileNum] = File{
			Name:    fn,
			Content: content,
		}
		fileNum++
	}
	return
}

func DirsForFiles(files []File) (dirs []string) {
	for _, file := range files {
		dirParts := strings.Split(filepath.Dir(file.Name), "/")
		for i := range dirParts {
			curDur := strings.Join(dirParts[:i+1], "/")
			if curDur == "." {
				continue
			}
			if arrayContains(dirs, curDur) {
				continue
			}
			dirs = append(dirs, curDur)
		}
	}
	return
}

func arrayContains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}
