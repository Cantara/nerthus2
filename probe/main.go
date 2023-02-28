package main

import (
	"bytes"
	"context"
	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/webserver"
	"github.com/cantara/gober/websocket"
	"github.com/cantara/nerthus2/message"
	jsoniter "github.com/json-iterator/go"
	"io"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"sync"
)

var json = jsoniter.ConfigFastest

func main() {
	portString := os.Getenv("webserver.port")
	port, err := strconv.Atoi(portString)
	if err != nil {
		log.WithError(err).Fatal("while getting webserver port")
	}
	serv, err := webserver.Init(uint16(port))
	if err != nil {
		log.WithError(err).Fatal("while initializing webserver")
	}
	NerthusConnector(context.Background())
	serv.Run()
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

type AnsibleTaskStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AnsibleAction struct {
	Playbook  string                   `json:"playbook"`
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
	defer os.Remove(f.Name())
	_, err = f.WriteString(action.Playbook)
	if err != nil {
		log.WithError(err).Fatal("unable to write tmp playbook")
	}

	exec := execute.NewDefaultExecute(
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
		Exec:           exec,
		StdoutCallback: "json",
		Options:        ansiblePlaybookOptions,
	}

	err = pb.Run(context.TODO())
	if err != nil {
		log.WithError(err).Error("while running ansible playbook")
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

func ActionHandler(action message.Action) (resp message.Response) {
	switch action.Action {
	case message.RoleUpdate:
	case message.Playbook:
		result := make(chan AnsibleTaskStatus)
		go func() {
			for status := range result {
				log.Info(status.Name, "status", status.Status)
			}
		}()
		AnsibleExecutor(AnsibleAction{
			Playbook:  action.AnsiblePlaybook,
			ExtraVars: action.ExtraVars,
			Results:   result,
		})
	}
	return
}

func NerthusConnector(ctx context.Context) {
	u, err := url.Parse("ws://" + os.Getenv("nerthus.url") + "/probe/testHost")
	if err != nil {
		log.WithError(err).Fatal("while parsing url to nerthus", "url", os.Getenv("nerthus.url"))
	}

	reader, writer, err := websocket.Dial[message.Action](u, ctx)
	if err != nil {
		log.WithError(err).Fatal("while connecting to nerthus", "url", u.String())
	}
	defer close(writer)
	for action := range reader {
		resp := ActionHandler(action)
		action.Response = &resp

		errChan := make(chan error, 1)
		select {
		case <-ctx.Done():
			return
		case writer <- websocket.Write[message.Action]{
			Data: action,
			Err:  errChan,
		}:
			err := <-errChan
			if err != nil {
				log.WithError(err).Error("unable to write response to nerthus", "response", resp, "action", action,
					"url", u.String(), "response_type", reflect.TypeOf(resp), "action_type", reflect.TypeOf(action))
				continue
			}
		}
	}
}
