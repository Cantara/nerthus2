package ansible

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	data, err := os.ReadFile("ansible/base.yml")
	if err != nil {
		panic(err)
	}
	var yml Playbook
	err = yaml.Unmarshal(data, &yml)
	if err != nil {
		panic(err)
	}
	out, err := yaml.Marshal(yml)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
	os.WriteFile("ansible/out.yml", out, 0644)
}
