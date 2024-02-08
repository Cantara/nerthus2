package main

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	log "github.com/cantara/bragi/sbragi"
)

type AnsibleTaskStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AnsibleAction struct {
	Playbook  []byte                   `json:"playbook"`
	ExtraVars map[string]string        `json:"extra_vars"`
	Results   chan<- AnsibleTaskStatus `json:"results"`
}

func AnsibleExecutor(action AnsibleAction) {
	defer close(action.Results)
	buff := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buff)
	f, err := os.CreateTemp("./", "playbook-*.yml")
	if err != nil {
		log.WithError(err).Fatal("unable to create tmp file for playbook")
	}
	//defer os.Remove(f.Name())
	_, err = f.Write(action.Playbook)
	if err != nil {
		log.WithError(err).Fatal("unable to write tmp playbook")
	}

	executor := execute.NewDefaultExecute(
		execute.WithWrite(io.Writer(buff)),
	)

	extraVarsConv := make(map[string]any)
	for k, v := range action.ExtraVars {
		extraVarsConv[k] = v
	}

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		ExtraVars: extraVarsConv,
	}

	pb := &playbook.AnsiblePlaybookCmd{
		Playbooks:      []string{f.Name()},
		Exec:           executor,
		StdoutCallback: "json",
		Options:        ansiblePlaybookOptions,
	}
	c, e := pb.Command()
	log.WithError(e).Info("playbook", "command", c)

	err = pb.Run(context.TODO())
	if err != nil {
		log.WithError(err).Error("while running ansible playbook", "name", f.Name())
		return
	}

	res, err := results.ParseJSONResultsStream(io.Reader(buff))
	if err != nil {
		panic(err)
	}

	for _, play := range res.Plays {
		for _, task := range play.Tasks {
			for _, content := range task.Hosts {
				//log.Info(task.Task.Name, "content", content)
				status := "Finished"
				if content.Changed {
					status = "Changed"
				} else if content.Failed {
					status = "Failed"
				} else if content.Skipped {
					status = "Skipped: " + content.SkipReason
				}
				action.Results <- AnsibleTaskStatus{
					Name:   task.Task.Name,
					Status: status,
				}
			}
		}
	}
}
