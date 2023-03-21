package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/syncmap"
	"github.com/cantara/gober/webserver"
	"github.com/cantara/gober/websocket"
	"github.com/cantara/nerthus2/ansible"
	"github.com/cantara/nerthus2/message"
	"github.com/cantara/nerthus2/system/service"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed roles bootstrap
var EFS embed.FS

var bootstrap bool
var gitRepo string
var gitToken string
var bootstrapEnv string

func init() {
	const ( //TODO: Add bootstrap git as a separate command from bootstrap.
		defaultBootstrap  = false
		bootstrapUsage    = "tells nerthus to bootstrap itself into aws"
		defaultGitRepo    = "github.com/cantara/nerthus2"
		gitRepoUsage      = "github repository for solution config"
		defaultGitToken   = ""
		gitTokenUsage     = "github repository granular access token"
		defaultSystemName = "exoreaction_demo"
		systemNameUsage   = "defines the system that Nerthus should use to provision itself"
	)
	flag.BoolVar(&bootstrap, "bootstrap", defaultBootstrap, bootstrapUsage)
	flag.BoolVar(&bootstrap, "b", defaultBootstrap, bootstrapUsage+" (shorthand)")
	flag.StringVar(&gitRepo, "git-repo", defaultGitRepo, gitRepoUsage)
	flag.StringVar(&gitRepo, "r", defaultGitRepo, gitRepoUsage+" (shorthand)")
	flag.StringVar(&gitToken, "git-token", defaultGitToken, gitTokenUsage)
	flag.StringVar(&gitToken, "t", defaultGitToken, gitTokenUsage+" (shorthand)")
	flag.StringVar(&bootstrapEnv, "environment", defaultSystemName, systemNameUsage)
	flag.StringVar(&bootstrapEnv, "e", defaultSystemName, systemNameUsage+" (shorthand)")
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func main() {
	flag.Parse()
	if bootstrap {
		_, err := GitCloneEnvironment(bootstrapEnv)
		if err != nil {
			log.WithError(err).Fatal("while cloning git repo during bootstrap")
		}
		Execute(bootstrapEnv)
		return
	}
	if gitToken == "" {
		gitToken = os.Getenv("git.token")
		gitRepo = os.Getenv("git.repo")
		bootstrapEnv = os.Getenv("env")
	}
	_, err := os.ReadDir("systems/" + bootstrapEnv)
	if err != nil {
		err = os.MkdirAll("systems", 0750)
		if err != nil {
			log.WithError(err).Fatal("while creating systems dir on first boot")
		}
		_, err = GitCloneEnvironment(bootstrapEnv)
		if err != nil {
			log.WithError(err).Fatal("while cloning git repo for bootstrapped environment")
		}
		Execute(bootstrapEnv)
	}
	/*
		bootstrap = true
		gitToken = "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"
		gitRepo = "github.com/SindreBrurberg/nerthus-test-config"
	*/
	portString := os.Getenv("webserver.port")
	port, err := strconv.Atoi(portString)
	if err != nil {
		log.WithError(err).Fatal("while getting webserver port")
	}
	serv, err := webserver.Init(uint16(port), true)
	if err != nil {
		log.WithError(err).Fatal("while initializing webserver")
	}

	serv.API.PUT("/environment/:env", func(c *gin.Context) {
	})

	serv.API.PUT("/provision/:artifactId", func(c *gin.Context) {
		artifactId := c.Param("artifactId")
		auth := webserver.GetAuthHeader(c)
		if auth == "" {
			webserver.ErrorResponse(c, "authorization not provided", http.StatusForbidden)
			return
		}
		if auth != os.Getenv("authkey") {
			webserver.ErrorResponse(c, "unauthorized", http.StatusUnauthorized)
			return
		}

		/*
			_, err := webserver.UnmarshalBody[service](c)
			if err != nil {
				webserver.ErrorResponse(c, err.Error(), http.StatusBadRequest)
				return
			}
		*/

		buff := bufPool.Get().(*bytes.Buffer)
		defer bufPool.Put(buff)

		exec := execute.NewDefaultExecute(
			execute.WithWrite(io.Writer(buff)),
		)

		ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
			ExtraVars: map[string]interface{}{
				"region":    "eu-west-3",             //"ap-northeast-1",
				"ami":       "ami-00575c0cbc20caf50", //"ami-0bba69335379e17f8",
				"cidr_base": "10.110.0",
				"service":   artifactId,
				"env":       "exoreaction-lab",
				"zone":      "lab.exoreaction.infra",
			},
		}

		pb := &playbook.AnsiblePlaybookCmd{
			Playbooks:      []string{"playbook.yml"},
			Exec:           exec,
			StdoutCallback: "json",
			Options:        ansiblePlaybookOptions,
		}

		err = pb.Run(context.TODO())
		if err != nil {
			log.WithError(err).Error("while running ansible playbook")
			webserver.ErrorResponse(c, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := results.ParseJSONResultsStream(io.Reader(buff))
		if err != nil {
			log.WithError(err).Fatal("while parsing json result stream")
		}

		msgOutput := struct {
			Host    string `json:"host"`
			Message string `json:"message"`
		}{}

		for _, play := range res.Plays {
			for _, task := range play.Tasks {
				for _, content := range task.Hosts {
					err = json.Unmarshal([]byte(fmt.Sprint(content.Stdout)), &msgOutput)
					if err != nil {
						panic(err)
					}

					fmt.Printf("[%s] %s\n", msgOutput.Host, msgOutput.Message)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "service added"})
		return
	})

	serv.API.PUT("/deploy/:artifactId", func(c *gin.Context) {
	})

	/*
			hc := make(chan message.Action, 10)
			hostActions.Set("testHost", hc)
			hc <- message.Action{
				Action: message.Playbook,
				AnsiblePlaybook: `---
		- name: Test playbook
		  hosts: localhost
		  connection: local
		  tasks:
		    - name: Ansible | Print test
		      debug:
		        msg: "test print"
		    - name: Ansible | Skipp me
		      debug:
		        msg: "test print"
		      when: false
		`,
				ExtraVars: nil,
			}
	*/

	websocket.Serve[message.Action](serv.API, "/probe/:host", nil, func(reader <-chan message.Action, writer chan<- websocket.Write[message.Action], p gin.Params, ctx context.Context) {
		host := p.ByName("host")
		log.Info("opening websocket", "host", host)
		defer log.Info("closed websocket", "host", host)
		go func() {
			hostChan, ok := hostActions.Get(host)
			if !ok {
				hostChan = make(chan message.Action, 10)
				hostActions.Set(host, hostChan)
			}
			for a := range hostChan {
				errChan := make(chan error, 1)
				action := websocket.Write[message.Action]{
					Data: a,
					Err:  errChan,
				}
				select {
				case <-ctx.Done():
					return
				case writer <- action:
					err := <-errChan
					if err != nil {
						log.WithError(err).Error("unable to write action to nerthus probe",
							"action_type", reflect.TypeOf(action))
						continue
					}
				}
			}
		}()

		for msg := range reader {
			if msg.Response == nil {
				log.Warning("read action response without response", "action", msg)
				continue
			}
			log.Info("response from action", "message", msg.Response.Message, "status", msg.Response.Status)
		}
	})

	serv.Run()
}

var hostActions = syncmap.New[chan message.Action]()

func ArrayContains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func GetService(u *url.URL) (serv service.Service, err error) {
	log.Trace("GetService", "url", u.String())
	resp, err := http.Get(u.String())
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("get miss")
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.WithError(err).Debug("while closing response body")
		}
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, &serv)
	return
}

type Condition struct {
	Field  string   `json:"Field"`
	Values []string `json:"Values"`
}
type Action struct {
	TargetGroupName string `json:"TargetGroupName"`
	Type            string `json:"Type"`
}

type Rule struct {
	Conditions []Condition `json:"Conditions"`
	Actions    []Action    `json:"Actions"`
	Priority   int         `json:"Priority"`
}

func ReadRoleDir(dir fs.FS, path string, roles map[string]ansible.Role) error {
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
		var role ansible.Role
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

// dir, err := os.MkdirTemp(".", "config-repo")
//
//	if err != nil {
//		log.WithError(err).Fatal("while creating tmpdir for git clone")
//	}
//
// defer os.RemoveAll(dir)

func GitAuth() *gitHttp.BasicAuth {
	return &gitHttp.BasicAuth{ //This is so stupid, but what GitHub wants
		Username: "nerthus",
		Password: gitToken,
	}
}

func GitCloneEnvironment(env string) (r *git.Repository, err error) {
	// Clones the repository into the given dir, just as a normal git clone does
	r, err = git.PlainClone("systems/"+env, false, &git.CloneOptions{
		Auth: GitAuth(),
		URL:  fmt.Sprintf("https://%s.git", gitRepo),
	})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			r, err = git.PlainOpen("systems/" + env)
			if err != nil {
				return
			}
			err = r.Fetch(&git.FetchOptions{
				Auth: GitAuth(),
			})
			if errors.Is(err, git.NoErrAlreadyUpToDate) {
				err = nil
			}
			return
		}
		return
	}
	return
}

func GitBootstrap(r *git.Repository, env string) {
	w, err := r.Worktree()
	if err != nil {
		log.WithError(err).Fatal("while getting git work tree")
	}
	err = fs.WalkDir(EFS, "bootstrap", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "bootstrap" {
			return nil
		}

		filename := strings.TrimPrefix(path, "bootstrap/")
		fullFilename := filepath.Join(env, filename)
		log.Info("processing file from EFS", "filename", filename)

		if d.IsDir() {
			err = os.Mkdir(fullFilename, 0750)
			if errors.Is(err, os.ErrExist) {
				return nil
			}
			return err
		}

		data, err := EFS.ReadFile(path)
		if err != nil {
			log.WithError(err).Fatal("while reading file from EFS")
		}
		err = os.WriteFile(fullFilename, data, 0640)
		if err != nil {
			log.WithError(err).Fatal("while writing file from EFS to gitrepo")
		}
		_, err = w.Add(filename)
		if err != nil {
			log.WithError(err).Fatal("while adding file to commit")
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Fatal("while walking bootstrap dir")
	}

	_, err = w.Commit("committing bootstrap", &git.CommitOptions{
		Author: &object.Signature{
			Name: "Nerthus",
			When: time.Now(),
		},
	})
	if err != nil {
		log.WithError(err).Fatal("while committing bootstrap")
	}

	err = r.Push(&git.PushOptions{
		Auth: GitAuth(),
	})
	if err != nil {
		log.WithError(err).Fatal("while pushing")
	}
}
