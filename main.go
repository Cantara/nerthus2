package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	amzaws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/consensus"
	"github.com/cantara/gober/discovery/local"
	"github.com/cantara/gober/eventmap"
	"github.com/cantara/gober/stream"
	"github.com/cantara/gober/stream/event"
	"github.com/cantara/gober/stream/event/store/inmemory"
	"github.com/cantara/gober/stream/event/store/ondisk"
	syncExt "github.com/cantara/gober/sync"
	"github.com/cantara/gober/webserver"
	"github.com/cantara/gober/webserver/health"
	"github.com/cantara/gober/websocket"
	"github.com/cantara/nerthus2/aws"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers"
	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/server"
	"github.com/cantara/nerthus2/message"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-git/go-git/v5"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

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

type environment struct {
	GitToken    string `json:"git_token"`
	GitRepo     string `json:"git_repo"`
	EnvName     string `json:"name"`
	lastExecute *time.Timer
}

func main() {
	flag.Parse()
	err := os.MkdirAll("systems", 0750)
	if err != nil {
		log.WithError(err).Fatal("while creating systems dir on boot")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var deploymentStream stream.Stream
	if bootstrap {
		deploymentStream, err = inmemory.Init("deployments", ctx)
	} else {
		deploymentStream, err = ondisk.Init("deployments", ctx)
	}
	if err != nil {
		log.WithError(err).Fatal("while initializing fairytale stream")
	}
	portString := os.Getenv("webserver.port")
	port, err := strconv.Atoi(portString)
	if err != nil {
		log.WithError(err).Fatal("while getting webserver port")
	}
	p, err := consensus.Init(uint16(port+1), os.Getenv("consensus.token"), local.New())
	if err != nil {
		log.WithError(err).Fatal("initing consensus")
	}
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.WithError(err).Fatal("while getting aws config")
	}
	cfg.RetryMode = amzaws.RetryModeAdaptive
	cfg.RetryMaxAttempts = 5
	e2, elb, rc, ac := ec2.NewFromConfig(cfg), elbv2.NewFromConfig(cfg), route53.NewFromConfig(cfg), acm.NewFromConfig(cfg)
	d, err := workers.New(deploymentStream, p.AddTopic, os.Getenv("deployment.crypto.key"), hostActions, e2, elb, rc, ac, ctx)
	if err != nil {
		log.WithError(err).Fatal("creating worker")
	}
	for i := 0; i < 100; i++ {
		go d.Work()
	}
	var envStream stream.Stream
	if bootstrap {
		envStream, err = inmemory.Init("environments", ctx)
	} else {
		envStream, err = ondisk.Init("environments", ctx)
	}
	if err != nil {
		log.WithError(err).Fatal("while initializing public key stream")
	}
	environments, err := eventmap.Init[environment](envStream, "environment", "v0.0.1",
		stream.StaticProvider(log.RedactedString(os.Getenv("environments.static_key"))), ctx, func(t event.Type, env *environment) {
			if t != event.Created {
				return
			}
			if env.EnvName == "" {
				return
			}
			log.Info("registered env", "name", env.EnvName)
			env.lastExecute = time.AfterFunc(time.Minute*15, func() {
				log.Info("-------timer executed new func--------")
				_, err := GitCloneEnvironment(*env)
				if log.WithError(err).Error("while cloning git repo during environment execution", "env", env.EnvName) {
					return
				}
				ExecuteEnv(env.EnvName, d.Provision)
			})
		})
	//TODO: If bootstrap add local users ssh key to map and add key on bootstrap

	if bootstrap {
		env := environment{
			GitToken: gitToken,
			GitRepo:  gitRepo,
			EnvName:  bootstrapEnv,
		}
		err = environments.Set(bootstrapEnv, env)
		if err != nil {
			log.WithError(err).Fatal("while storing bootstrap env in map", "isBootstrapping", bootstrap, "environments", environments.Len())
		}
		_, err := GitCloneEnvironment(env)
		if err != nil {
			log.WithError(err).Fatal("while cloning git repo during bootstrap")
		}
		ExecuteEnv(bootstrapEnv, d.Provision)
		return
	}
	if environments.Len() == 0 {
		err = environments.Set(os.Getenv("boot_env"), environment{
			GitToken: os.Getenv("git.token"),
			GitRepo:  os.Getenv("git.repo"),
			EnvName:  os.Getenv("boot_env"),
		})
		if err != nil {
			log.WithError(err).Fatal("while storing bootstrap env in map", "isBootstrapping", bootstrap, "environments", environments.Len())
		}
	}
	serv, err := webserver.Init(uint16(port), true)
	if err != nil {
		log.WithError(err).Fatal("while initializing webserver")
	}

	configEnv := func(c *gin.Context) {
		name := c.Params.ByName("env")
		env, err := environments.Get(name)
		//if ok := environments.Exists(env); !ok {
		if err != nil {
			c.AbortWithStatus(404)
			return
		}
		//_, err := GitCloneEnvironment(env, environments)
		_, err = GitCloneEnvironment(env)
		if err != nil {
			log.WithError(err).Fatal("while cloning git repo during environment execution", "env", env)
		}
		env.lastExecute.Reset(time.Minute * 15)
		go ExecuteEnv(env.EnvName, d.Provision)
		/*
			t := time.NewTicker(time.Second * 30)
			for {
				select {
				case <-c.Request.Context().Done():
					return
				case <-t.C:
					c.SSEvent("ping", nil)
				case result, ok := <-resultChan:
					if !ok {
						return
					}
					out, _ := jsoniter.ConfigFastest.Marshal(result)
					c.SSEvent("result", string(out))
				}
			}
		*/
	}
	serv.API().POST("/config/:env", configEnv)

	serv.API().PUT("/config/:env", configEnv)

	/*
		serv.API().PUT("/config/:env/:sys", func(c *gin.Context) {
			env := c.Params.ByName("env")
			if ok := environments.Exists(env); !ok {
				c.AbortWithStatus(404)
				return
			}
			sys := c.Params.ByName("sys")
			_, err := GitCloneEnvironment(env, environments)
			if err != nil {
				log.WithError(err).Fatal("while cloning git repo during system execution", "env", env, "system", sys)
			}
			go ExecuteSys(env, sys, d.ProvisionSystem)
			/*
				resultChan := make(chan string)
				t := time.NewTicker(time.Second * 30)
				for {
					select {
					case <-c.Request.Context().Done():
						return
					case <-t.C:
						c.SSEvent("ping", nil)
					case result, ok := <-resultChan:
						if !ok {
							return
						}
						out, _ := jsoniter.ConfigFastest.Marshal(result)
						c.SSEvent("result", string(out))
					}
				}
	*/
	//})

	/*
		serv.API().PUT("/config/:env/:sys/:cluster", func(c *gin.Context) {
			env := c.Params.ByName("env")
			if ok := environments.Exists(env); !ok {
				log.Warning("put aborted", "env", env, "envs", environments.Keys())
				c.AbortWithStatus(404)
				return
			}
			sys := c.Params.ByName("sys")
			cluster := c.Params.ByName("cluster")
			_, err := GitCloneEnvironment(env, environments)
			if err != nil {
				log.WithError(err).Fatal("while cloning git repo during service execution", "env", env, "system", sys, "cluster", cluster)
			}
			go ExecuteCluster(env, sys, cluster, d.ProvisionCluster)
		})

		serv.API().PUT("/config/:env/:sys/:cluster/:serv", func(c *gin.Context) {
			env := c.Params.ByName("env")
			if ok := environments.Exists(env); !ok {
				log.Warning("put aborted", "env", env, "envs", environments.Keys())
				c.AbortWithStatus(404)
				return
			}
			sys := c.Params.ByName("sys")
			cluster := c.Params.ByName("cluster")
			service := c.Params.ByName("serv")
			_, err := GitCloneEnvironment(env, environments)
			if err != nil {
				log.WithError(err).Fatal("while cloning git repo during service execution", "env", env, "system", sys, "cluster", cluster, "service", service)
			}

			ExecuteServ(env, sys, cluster, service)
		})
	*/

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
		auth := serv.API().Group("")
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
		auth.PUT("/ssh/:server", func(c *gin.Context) {
			name := c.Params.ByName("server")
			servs, err := server.GetServers(name, e2)
			if err != nil {
				if errors.Is(err, server.ErrServerNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
				} else {
					log.WithError(err).Error("while getting servers from aws")
					c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				}
				return
			}
			g, err := security.ById(servs[0].SecutityGroupId, e2)
			if err != nil {
				log.WithError(err).Error("while getting server security group from aws")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			}
			err = g.OpenSSH("TMP_ALL_USER", c.ClientIP(), e2)
			if err != nil {
				log.WithError(err).Error("while opening ssh to server")
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			}

			c.JSON(http.StatusOK, nil)
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

		auth.GET("/env", func(c *gin.Context) {
			c.JSON(http.StatusOK, environments.Keys())
		})

		auth.PUT("/env/:env", func(c *gin.Context) {
			var env environment
			err := c.MustBindWith(&env, binding.JSON)
			if err != nil {
				log.WithError(err).Debug("while binding json body from key put")
				return
			}
			if env.EnvName != c.Params.ByName("env") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "name does not match name of env", "name": c.Params.ByName("name"), "env": env})
				return
			}
			err = environments.Set(env.EnvName, env)
			if err != nil {
				log.WithError(err).Error("while storing env in map", "environments", environments.Keys())
				c.JSON(http.StatusInternalServerError, gin.H{"error": "while storing env in map"})
				return
			}
			configEnv(c)
		})
	}

	websocket.Serve(serv.API(), "/probe/:host", nil, func(reader <-chan message.Action, writer chan<- websocket.Write[message.Action], p gin.Params, ctx context.Context) {
		defer close(writer)
		host := p.ByName("host")
		log.Info("opening websocket", "host", host)
		defer log.Info("closed websocket", "host", host)

		var authorizedKeys bytes.Buffer
		for _, k := range keyMap.GetMap() {
			authorizedKeys.WriteString(k.Data)
			authorizedKeys.WriteRune('\n')
		}
		b := authorizedKeys.Bytes()

		errChan := make(chan error, 1)
		writer <- websocket.Write[message.Action]{
			Data: message.Action{
				Action: message.AuthorizedKeys,
				Data:   b,
			},
			Err: errChan,
		}
		err := <-errChan
		if err != nil {
			log.WithError(err).Error("unable to write action to nerthus probe")
			return //TODO continue
		}

		hostChan, ok := hostActions.Get(host)
		if !ok {
			hostChan = make(chan message.Action, 10)
			hostActions.Set(host, hostChan)
		}
	Worker:
		for {
			select {
			case <-ctx.Done():
				break Worker
			case msg, ok := <-reader:
				if !ok {
					break Worker
				}
				if msg.Response == nil {
					log.Warning("read action response without response", "action", msg)
					continue Worker
				}
				log.Info("response from action", "message", msg.Response.Message, "status", msg.Response.Status)
			case a := <-hostChan:
				errChan := make(chan error, 1)
				action := websocket.Write[message.Action]{
					Data: a,
					Err:  errChan,
				}
				select {
				case <-ctx.Done():
					break Worker
				case writer <- action:
					err := <-errChan
					if err != nil {
						log.WithError(err).Error("unable to write action to nerthus probe",
							"action_type", reflect.TypeOf(action))
						continue Worker
					}
				}
			}
		}
		log.Info("reader closed, ending websocket function")
	})

	//https://visuale.greps.dev/api/status/prod/Stamp-server/prod-greps-stamp-server?service_tag=Greps&service_type=A2A
	visuale := make(map[string]map[string]map[string][]health.Report)
	var vLock sync.Mutex
	serv.Base().PUT("/api/status/:env/:service/:hostname", func(c *gin.Context) {
		env := c.Params.ByName("env")
		service := c.Params.ByName("service")
		hostname := c.Params.ByName("hostname")

		var report health.Report
		err := c.MustBindWith(&report, binding.JSON)
		if err != nil {
			log.WithError(err).Debug("while binding json body from key put")
			return
		}

		vLock.Lock()
		defer vLock.Unlock()

		if _, ok := visuale[env]; !ok {
			visuale[env] = make(map[string]map[string][]health.Report)
		}
		if _, ok := visuale[env][service]; !ok {
			visuale[env][service] = make(map[string][]health.Report)
		}
		if _, ok := visuale[env][service][hostname]; !ok {
			visuale[env][service][hostname] = make([]health.Report, 0)
		}
		reports := visuale[env][service][hostname]
		reports = append(reports, report)
		visuale[env][service][hostname] = reports

		if strings.ToLower(report.Status) != strings.ToLower(reports[len(reports)-2].Status) {
			log.Warning("node changed visuale status", "env", env, "service", service, "hostname", hostname, "status", report.Status)
		}
	})

	go func() {
		for t := range time.NewTicker(time.Second).C {
			if t.Second()%30 == 0 {
				vLock.Lock()
				d, err := json.Marshal(visuale)
				for env := range visuale {
					for service := range visuale[env] {
						for hostname, reports := range visuale[env][service] {
							visuale[env][service][hostname] = reports[len(reports)-1:]
						}
					}
				}
				vLock.Unlock()
				if err != nil {
					log.WithError(err).Error("while marshalling json")
					continue
				}
				f, err := os.OpenFile("visuale.json", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640)
				if err != nil {
					log.WithError(err).Error("while opening visuale.json file")
					continue
				}
				_, err = f.WriteString(string(d))
				if err != nil {
					log.WithError(err).Error("while writing visuale.json file")
				}
				f.Close()
			}
		}
	}()

	log.Info("starting webserver", "environments", environments.Keys())
	go p.Run()
	serv.Run()
}

var hostActions = syncExt.NewMap[chan message.Action]()

func GitAuth(gitConf environment) *gitHttp.BasicAuth {
	return &gitHttp.BasicAuth{ //This is so stupid, but what GitHub wants
		Username: "nerthus",
		Password: gitConf.GitToken,
	}
}

//var ErrEnvNotFound = errors.New("environment not found")

func GitCloneEnvironment(env environment) (r *git.Repository, err error) {
	// Clones the repository into the given dir, just as a normal git clone does
	/*
		gitConf, err := environments.Get(env)
		if err != nil {
			//err = ErrEnvNotFound
			return
		}
	*/
	r, err = git.PlainClone("systems/"+env.EnvName, false, &git.CloneOptions{
		Auth: GitAuth(env),
		URL:  fmt.Sprintf("https://%s.git", env.GitRepo),
	})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			r, err = git.PlainOpen("systems/" + env.EnvName)
			if err != nil {
				return
			}
			var w *git.Worktree
			w, err = r.Worktree()
			if err != nil {
				return
			}
			err = w.Pull(&git.PullOptions{Auth: GitAuth(env)})
			if errors.Is(err, git.NoErrAlreadyUpToDate) {
				err = nil
			}
			return
		}
		return
	}
	return
}

/*
func GitBootstrap(r *git.Repository, env string, gitConf properties.BootstrapVars) {
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
*/

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
