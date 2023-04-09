package executor

import (
	"bytes"
	"context"
	"fmt"
	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	log "github.com/cantara/bragi/sbragi"
	"io"
	"os"
	"sync"
)

type TaskResult struct {
	Err     error  `json:"error"`
	Name    string `json:"task_name"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Command string `json:"command"`
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func Execute(playbookPath string, extraVars map[string]any, ctx context.Context) (resultChan <-chan TaskResult) {
	buff := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buff)

	exec := execute.NewDefaultExecute(
		execute.WithWrite(io.Writer(buff)),
	)

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		ExtraVars: extraVars,
	}

	pb := &playbook.AnsiblePlaybookCmd{
		Playbooks:      []string{playbookPath}, //dir + "/ansible/" + play},
		Exec:           exec,
		StdoutCallback: "json",
		Options:        ansiblePlaybookOptions,
	}

	out := make(chan TaskResult, 5)
	resultChan = out
	go func() {
		defer close(out)
		playbookError := pb.Run(ctx)

		res, err := results.ParseJSONResultsStream(io.Reader(buff))
		if err != nil {
			log.WithError(err).Fatal("while parsing json result stream") //Don't think this should happen
		}

		for _, play := range res.Plays {
			for _, task := range play.Tasks {
				for _, content := range task.Hosts {
					result := TaskResult{
						Name:    task.Task.Name,
						Message: fmt.Sprint(content.Msg),
						Command: fmt.Sprint(content.Cmd),
					}
					if content.Changed {
						result.Status = "Changed"
					} else if content.Failed {
						result.Status = "Failed"
						result.Err = playbookError
					} else if content.Skipped {
						result.Status = "Skipped: " + content.SkipReason
					} else {
						result.Status = "Finished"
					}
					log.Debug(result.Name, "status", result.Status, "output", fmt.Sprint(content.Msg))
					out <- result
					if result.Err != nil {
						return
					}
				}
			}
		}
	}()
	return
}

func WriteNodePlay(nodeDir, nodeName string, play []byte, bootstrap bool) (err error) {
	os.Mkdir(nodeDir, 0750)
	if bootstrap {
		nodeName = fmt.Sprintf("%s_bootstrap", nodeName)
	}
	fn := fmt.Sprintf("%s/%s.yml", nodeDir, nodeName)
	os.Remove(fn)
	err = os.WriteFile(fn, play, 0640)
	if err != nil {
		return
	}
	return
}
