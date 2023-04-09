package executors

import (
	"context"
	"fmt"
	ansibleExecutor "github.com/cantara/nerthus2/executors/ansible/executor"
	"os"
	"strings"
)

func ExecuteLoadbalancer(dir string, vars map[string]any, ctx context.Context) (resultChan <-chan ansibleExecutor.TaskResult) {
	return ansibleExecutor.Execute(FindPlayPath(dir, "loadbalancer.yml"), vars, ctx)
}

func FindPlayPath(dir, play string) (out string) {
	parts := strings.Split(dir, "/")
	for i := len(parts); i > 0; i -= 2 {
		out = fmt.Sprintf("%s/ansible/%s", strings.Join(parts[:i], "/"), play)
		_, err := os.Stat(out)
		if err != nil {
			continue
		}
		return out
	}
	return ""
}
