package executors

import (
	"context"
	"github.com/cantara/nerthus2/executors/ansible/executor"
)

func ExecuteService(dir string, vars map[string]any, ctx context.Context) (resultChan <-chan executor.TaskResult) {
	return executor.Execute(FindPlayPath(dir, "provision.yml"), vars, ctx)
}
