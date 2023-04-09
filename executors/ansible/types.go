package ansible

import (
	"errors"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"strings"
)

type Playbook struct {
	Name       string           `yaml:"name"`
	Hosts      string           `yaml:"hosts"`
	Connection string           `yaml:"connection"`
	Vars       map[string]any   `yaml:"vars"`
	Tasks      []map[string]any `yaml:"tasks"`
}

type Role struct {
	Id           string
	Name         string            `yaml:"name"`
	Dependencies []Dependency      `yaml:"dependencies"`
	Vars         map[string]string `yaml:"vars"`
	Tasks        []map[string]any  `yaml:"tasks"`
}

type Dependency struct {
	Role string `yaml:"role"`
}

type Tasks map[string]any

type SecurityGroupRule struct {
	Proto    string `yaml:"proto" json:"proto"`
	FromPort string `yaml:"from_port" json:"from_port"`
	ToPort   string `yaml:"to_port" json:"to_port"`
	Group    string `yaml:"group_name" json:"group_name"`
}

func ReadRoleDir(dir fs.FS, path string, roles map[string]Role) error {
	_, err := dir.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return fs.WalkDir(dir, path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(dir, path)
		if err != nil {
			return err
		}
		var role Role
		err = yaml.Unmarshal(b, &role)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(d.Name(), ".yml")
		role.Id = name
		roles[name] = role
		return nil
	})
}
