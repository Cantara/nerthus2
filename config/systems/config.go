package systems

import (
	"gopkg.in/yaml.v3"
	"io/fs"
)

func LoadConfig[T any](curFS fs.FS) (out T, err error) {
	data, err := fs.ReadFile(curFS, "config.yml")
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return
	}
	return
}
