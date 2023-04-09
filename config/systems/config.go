package systems

import (
	"gopkg.in/yaml.v3"
	"os"
)

func LoadConfig[T any](dir string) (out T, err error) {
	data, err := os.ReadFile(dir + "/config.yml")
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return
	}
	return
}
