package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/stream"
	"github.com/cantara/gober/webserver"
	"github.com/cantara/gober/websocket"
	"github.com/cantara/nerthus2/message"
	"github.com/cantara/nerthus2/probe/actions"
	"github.com/cantara/nerthus2/probe/config"
	"github.com/cantara/nerthus2/probe/statemachine"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigFastest
var wd, _ = os.Getwd()

func main() {
	actions.DryRun = false
	actions.IsRoot = os.Geteuid() == 0 && os.Getenv("USER") == "root" &&
		os.Getenv("SUDO_UID") != "" && os.Getenv("SUDO_GID") != "" &&
		os.Getenv("SUDO_USER") != "" && os.Getenv("HOME") != ""
	if !actions.IsRoot {
		log.Trace("root calculations", "isroot", actions.IsRoot, "euid", os.Geteuid(), "user", os.Getenv("USER"), "args[0]", os.Args[0], "suid", os.Getenv("SUDO_UID"), "sgid", os.Getenv("SUDO_GID"), "suser", os.Getenv("SUDO_USER"), "home", os.Getenv("HOME"))
		fmt.Println("NOT running under sudo. Please execute the following command instead")
		fmt.Printf("sudo %s\n", strings.Join(os.Args, " "))
		return
	}
	username := os.Getenv("SUDO_USER")
	u, err := user.Lookup(username)
	log.WithError(err).Fatal("getting system user")
	if u.Username != username {
		log.Fatal("Username for systemuser was not same as provided by sudo")
	}
	if u.Uid != os.Getenv("SUDO_UID") {
		log.Fatal("UID for systemuser was not same as provided by sudo")
	}
	if u.Gid != os.Getenv("SUDO_GID") {
		log.Fatal("GID for systemuser was not same as provided by sudo")
	}
	/* Not sure i care
	if u.HomeDir != os.Getenv("HOME") {
		log.Fatal("Home for systemuser was not same as provided by sudo", "homedir", u.HomeDir, "home", os.Getenv("HOME"), "?", u.HomeDir == os.Getenv("HOME"))
	}
	*/
	uid, err := strconv.Atoi(u.Uid)
	log.WithError(err).Fatal("getting systemuser uid as int")
	gid, err := strconv.Atoi(u.Gid)
	log.WithError(err).Fatal("getting systemuser gid as int")
	actions.Users[username] = actions.User{
		Username: username,
		UID:      uid,
		GID:      gid,
		Home:     u.HomeDir,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sm, err := statemachine.New[config.Environment]("TestStateMaching_", stream.StaticProvider(log.RedactedString(os.Getenv("deployment.crypto.key"))), "next", ctx)
	log.WithError(err).Fatal("while initializing statemachine")
	sm.Func("next", actions.Next)
	sm.Func(actions.Execute("install", actions.Install))
	sm.Func(actions.Execute("download", actions.Download))
	sm.Func(actions.Execute("link", actions.Link))
	sm.Func(actions.Execute("enable", actions.Enable))
	sm.Func(actions.Execute("file_string", actions.FileString))
	sm.Func(actions.Execute("file_bytes", actions.FileBytes))
	sm.Func(actions.Execute("user", actions.Useradd))
	sm.Func(actions.Execute("command", actions.Command))

	portString := os.Getenv("webserver.port")
	port, err := strconv.Atoi(portString)
	if err != nil {
		log.WithError(err).Fatal("while getting webserver port")
	}
	serv, err := webserver.Init(uint16(port), false)
	if err != nil {
		log.WithError(err).Fatal("while initializing webserver")
	}
	go KeepRunning(func() {
		NerthusConnector(sm, context.Background())
	}, "nerthus connector")
	go sm.Run(ctx)
	serv.Run()
}

func KeepRunning(f func(), s string) {
	defer func() {
		r := recover()
		if r != nil {
			log.WithError(fmt.Errorf("recovered: %v", r)).Error("while keep running", "name", s)
			KeepRunning(f, s)
		}
	}()
	for {
		f()
	}
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

func ActionHandler(sm statemachine.StateMachine[config.Environment], action message.Action, resp chan<- websocket.Write[message.Action]) {
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
	case message.Config:
		var cfg config.Environment
		if err := json.Unmarshal(action.Data, &cfg); log.WithError(err).Error("read json config") {
			action.Response = &message.Response{
				Status:  "FAILED",
				Message: "while unmarshaling config",
				Error:   err,
			}
			resp <- websocket.Write[message.Action]{
				Data: action,
			}
			return
		}
		sm.Start(cfg)
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

func NerthusConnector(sm statemachine.StateMachine[config.Environment], ctx context.Context) {
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
		ActionHandler(sm, action, writer)
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
