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
	"os/exec"
	"strconv"
	"sync"
	"time"
)

var json = jsoniter.ConfigFastest
var wd, _ = os.Getwd()

func main() {
	portString := os.Getenv("webserver.port")
	port, err := strconv.Atoi(portString)
	if err != nil {
		log.WithError(err).Fatal("while getting webserver port")
	}
	serv, err := webserver.Init(uint16(port), false)
	if err != nil {
		log.WithError(err).Fatal("while initializing webserver")
	}
	go func() {
		for {
			NerthusConnector(context.Background())
		}
	}()
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
		log.WithError(err).Error("while running ansible playbook")
		exec.Command("sudo", "reboot").Run()
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

func ActionHandler(action message.Action, resp chan<- websocket.Write[message.Action]) {
	switch action.Action {
	case message.RoleUpdate:
	case message.Playbook:
		result := make(chan AnsibleTaskStatus)
		go func() {
			for status := range result {
				log.Info(status.Name, "status", status.Status)
				action.Response = &message.Response{
					Status:  status.Status,
					Message: status.Name,
				}
				resp <- websocket.Write[message.Action]{
					Data: action,
				}
			}
		}()
		AnsibleExecutor(AnsibleAction{
			Playbook:  action.Data,
			ExtraVars: action.ExtraVars,
			Results:   result,
		})
	case message.AuthorizedKeys:
		f, err := os.OpenFile(wd+"/.ssh/authorized_keys", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
		if err != nil {
			log.WithError(err).Error("while setting authorized keys")
			action.Response = &message.Response{
				Status:  "FAILED",
				Message: "while setting authorized keys",
				Error:   err,
			}
			resp <- websocket.Write[message.Action]{
				Data: action,
			}
		}
		err = WriteAll(f, action.Data)
		if err != nil {
			log.WithError(err).Error("while writing authorized keys to file")
			action.Response = &message.Response{
				Status:  "FAILED",
				Message: "while writing authorized keys to file",
				Error:   err,
			}
			resp <- websocket.Write[message.Action]{
				Data: action,
			}
		}
		action.Response = &message.Response{
			Status:  "SUCCESS",
			Message: "wrote authorized keys to file",
		}
		resp <- websocket.Write[message.Action]{
			Data: action,
		}
	}
	return
}

func WriteAll(w io.Writer, data []byte) (err error) {
	totalOut := 0
	var n int
	for totalOut < len(data) {
		n, err = w.Write(data[totalOut:])
		if err != nil {
			log.WithError(err).Error("while writing all")
			return
		}
		totalOut += n
	}
	return
}

func NerthusConnector(ctx context.Context) {
	uri := "wss://" + os.Getenv("nerthus.url") + "/probe/" + os.Getenv("hostname")
	u, err := url.Parse(uri)
	if err != nil {
		log.WithError(err).Fatal("while parsing url to nerthus", "url", uri)
	}

	reader, writer, err := websocket.Dial[message.Action](u, ctx)
	if err != nil {
		log.WithError(err).Error("while connecting to nerthus", "url", u.String())
		time.Sleep(15 * time.Second)
		return
	}
	defer close(writer)
	for action := range reader {
		ActionHandler(action, writer)
		//action.Response = &resp

		/*
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
					return //TODO: continue
				}
			}
		*/
	}
}
