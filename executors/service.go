package executors

import "context"

func ExecuteService(dir string, vars map[string]any, ctx context.Context) (resultChan <-chan TaskResult) {
	play := "provision.yml"
	return Execute(dir+"/ansible/"+play, vars, ctx)
}
