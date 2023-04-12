package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/eventmap"
	"github.com/cantara/gober/stream"
	"github.com/cantara/gober/stream/event/store/ondisk"
	"github.com/cantara/gober/syncmap"
	"github.com/cantara/gober/webserver"
	"github.com/cantara/gober/websocket"
	"github.com/cantara/nerthus2/aws"
	"github.com/cantara/nerthus2/config"
	"github.com/cantara/nerthus2/message"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
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

var environments = make(map[string]config.BootstrapVars)

func main() {
	flag.Parse()
	err := os.MkdirAll("systems", 0750)
	if err != nil {
		log.WithError(err).Fatal("while creating systems dir on boot")
	}
	if bootstrap {
		environments[bootstrapEnv] = config.BootstrapVars{
			GitToken: gitToken,
			GitRepo:  gitRepo,
			EnvName:  bootstrapEnv,
		}
		_, err := GitCloneEnvironment(bootstrapEnv)
		if err != nil {
			log.WithError(err).Fatal("while cloning git repo during bootstrap")
		}
		ExecuteEnv(bootstrapEnv)
		return
	}
	if gitToken == "" {
		environments[os.Getenv("env")] = config.BootstrapVars{
			GitToken: os.Getenv("git.token"),
			GitRepo:  os.Getenv("git.repo"),
			EnvName:  os.Getenv("env"),
		}
	}
	portString := os.Getenv("webserver.port")
	port, err := strconv.Atoi(portString)
	if err != nil {
		log.WithError(err).Fatal("while getting webserver port")
	}
	serv, err := webserver.Init(uint16(port), true)
	if err != nil {
		log.WithError(err).Fatal("while initializing webserver")
	}

	serv.API.PUT("/config/:env", func(c *gin.Context) {
		env := c.Params.ByName("env")
		if _, ok := environments[env]; !ok {
			c.AbortWithStatus(404)
			return
		}
		_, err := GitCloneEnvironment(env)
		if err != nil {
			log.WithError(err).Fatal("while cloning git repo during environment execution", "env", env)
		}
		ExecuteEnv(env)
	})

	serv.API.PUT("/config/:env/:sys", func(c *gin.Context) {
		env := c.Params.ByName("env")
		if _, ok := environments[env]; !ok {
			c.AbortWithStatus(404)
			return
		}
		sys := c.Params.ByName("sys")
		_, err := GitCloneEnvironment(env)
		if err != nil {
			log.WithError(err).Fatal("while cloning git repo during system execution", "env", env, "system", sys)
		}
		ExecuteSys(env, sys)
	})

	serv.API.PUT("/config/:env/:sys/:serv", func(c *gin.Context) {
		env := c.Params.ByName("env")
		if _, ok := environments[env]; !ok {
			log.Warning("put aborted", "env", env, "envs", keys(environments))
			c.AbortWithStatus(404)
			return
		}
		sys := c.Params.ByName("sys")
		serv := c.Params.ByName("serv")
		_, err := GitCloneEnvironment(env)
		if err != nil {
			log.WithError(err).Fatal("while cloning git repo during service execution", "env", env, "system", sys, "service", serv)
		}
		ExecuteServ(env, sys, serv)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	keyStream, err := ondisk.Init("pubKeys", ctx)
	if err != nil {
		log.WithError(err).Fatal("while initializing public key stream")
	}
	keyMap, err := eventmap.Init[key](keyStream, "pubkey", "v0.0.1",
		stream.StaticProvider(log.RedactedString(os.Getenv("pubkey.static_key"))), ctx)
	if err != nil {
		log.WithError(err).Fatal("while initializing public key event map")
	}
	{
		auth := serv.API.Group("")
		accounts := gin.Accounts{}
		accounts[os.Getenv("api.username")] = os.Getenv("api.password")
		auth.Use(gin.BasicAuth(accounts))

		auth.GET("/servers", func(c *gin.Context) {
			servers, err := aws.GetServers()
			if err != nil {
				log.WithError(err).Error("while getting servers from aws")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				return
			}
			c.JSON(http.StatusOK, servers)
		})

		auth.PUT("/key/:user/:name", func(c *gin.Context) {
			var ky key
			err := c.MustBindWith(&ky, binding.JSON)
			if err != nil {
				log.WithError(err).Debug("while binding json body from key put")
				return
			}
			if ky.Name != c.Params.ByName("name") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "name does not match name of key"})
				return
			}
			err = keyMap.Set(fmt.Sprintf("%s-%s", c.Params.ByName("user"), ky.Name), ky)
			if err != nil {
				log.WithError(err).Error("while storing new public key")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "error while storing new public key"})
				return
			}

			var authorizedKeys bytes.Buffer
			for _, k := range keyMap.GetMap() {
				authorizedKeys.WriteString(k.Data)
				authorizedKeys.WriteRune('\n')
			}
			b := authorizedKeys.Bytes()
			for _, srv := range hostActions.GetMap() {
				srv <- message.Action{
					Action: message.AuthorizedKeys,
					Data:   b,
				}
			}
		})
	}

	websocket.Serve[message.Action](serv.API, "/probe/:host", nil, func(reader <-chan message.Action, writer chan<- websocket.Write[message.Action], p gin.Params, ctx context.Context) {
		defer close(writer)
		host := p.ByName("host")
		log.Info("opening websocket", "host", host)
		defer log.Info("closed websocket", "host", host)
		go func() {
			for msg := range reader {
				if msg.Response == nil {
					log.Warning("read action response without response", "action", msg)
					continue
				}
				log.Info("response from action", "message", msg.Response.Message, "status", msg.Response.Status)
			}
		}()

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
					return //TODO continue
				}
			}
		}
		log.Info("reader closed, ending websocket function")
	})

	log.Info("starting webserver", "environments", keys(environments))
	serv.Run()
}

var hostActions = syncmap.New[chan message.Action]()

func GitAuth(gitConf config.BootstrapVars) *gitHttp.BasicAuth {
	return &gitHttp.BasicAuth{ //This is so stupid, but what GitHub wants
		Username: "nerthus",
		Password: gitConf.GitToken,
	}
}

var ErrEnvNotFound = errors.New("environment not found")

func GitCloneEnvironment(env string) (r *git.Repository, err error) {
	// Clones the repository into the given dir, just as a normal git clone does
	gitConf, ok := environments[env]
	if !ok {
		err = ErrEnvNotFound
		return
	}
	r, err = git.PlainClone("systems/"+env, false, &git.CloneOptions{
		Auth: GitAuth(gitConf),
		URL:  fmt.Sprintf("https://%s.git", gitConf.GitToken),
	})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			r, err = git.PlainOpen("systems/" + env)
			if err != nil {
				return
			}
			var w *git.Worktree
			w, err = r.Worktree()
			if err != nil {
				return
			}
			err = w.Pull(&git.PullOptions{Auth: GitAuth(gitConf)})
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
	gitConf, ok := environments[env]
	if !ok {
		return
	}
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
		Auth: GitAuth(gitConf),
	})
	if err != nil {
		log.WithError(err).Fatal("while pushing")
	}
}

func keys[T any](m map[string]T) (keys []string) {
	keys = make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return
}

type key struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

//var kys = map[string]key{}
