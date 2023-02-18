package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	"github.com/gin-gonic/gin"
	ws "nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	log "github.com/cantara/bragi"
	"github.com/cantara/gober/webserver"
)

type service struct {
}

func main() {
	portString := os.Getenv("webserver.port")
	port, err := strconv.Atoi(portString)
	if err != nil {
		log.AddError(err).Fatal("while getting webserver port")
	}
	serv, err := webserver.Init(uint16(port))
	if err != nil {
		log.AddError(err).Fatal("while initializing webserver")
	}
	bufPool := sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}
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
			log.AddError(err).Error("while running ansible playbook")
			webserver.ErrorResponse(c, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := results.ParseJSONResultsStream(io.Reader(buff))
		if err != nil {
			panic(err)
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

	Websocket(serv.API, "/probe/:host", nil, func(ctx context.Context, conn *ws.Conn, p gin.Params) {
		//conn.CloseRead(ctx)
		host := p.ByName("host")
		go func() {
			for a := range hostActions[host] {
				err = wsjson.Write(ctx, conn, a)
				if err != nil {
					log.AddError(err).Warning("while writing to socket")
					return
				}
			}

		}()

		var msg messagePackage[any]
		for err != nil {
			err = wsjson.Read(ctx, conn, &msg)
			if err != nil {
				continue
			}

		}
		return
	})

	serv.Run()
}

var hostActions map[string]chan action

type action struct {
	Playbook string `json:"playbook"`
}

type messagePackage[T any] struct {
	Type string `json:"type"`
	Data T      `json:"data"`
}

func Websocket(r *gin.RouterGroup, path string, acceptFunc func(c *gin.Context) bool, wsfunc func(ctx context.Context, conn *ws.Conn, params gin.Params)) {
	r.GET(path, func(c *gin.Context) {
		if acceptFunc != nil && !acceptFunc(c) {
			return //Could be smart to have some check of weather or not the statuscode code has been set.
		}
		s, err := ws.Accept(c.Writer, c.Request, nil)
		if err != nil {
			log.AddError(err).Fatal("while accepting websocket")
		}
		defer s.Close(ws.StatusNormalClosure, "") //Could be smart to do something here to fix / tell people of errors.
		wsfunc(c.Request.Context(), s, c.Params)
	})
}
