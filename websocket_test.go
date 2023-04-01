package main

import (
	"context"
	"fmt"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/webserver"
	"github.com/cantara/gober/websocket"
	"github.com/cantara/nerthus2/message"
	"github.com/gin-gonic/gin"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestWebsucker(t *testing.T) {
	serv, err := webserver.Init(4123, true)
	if err != nil {
		t.Error(err)
		return
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Second)
		hostChan, ok := hostActions.Get("test-probe")
		if !ok {
			hostChan = make(chan message.Action, 10)
			hostActions.Set("test-probe", hostChan)
		}
		for i := 0; i < 10; i++ {
			hostChan <- message.Action{
				Action:          fmt.Sprintf("test-action-%d", i),
				AnsiblePlaybook: make([]byte, 1024),
				ExtraVars:       nil,
				Response:        nil,
			}
			time.Sleep(250 * time.Millisecond)
		}
		//close(hostChan)
	}()
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
		log.Info("writing closed, ending websocket function")
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		uri := "ws://localhost:4123/probe/test-probe"
		u, err := url.Parse(uri)
		if err != nil {
			log.WithError(err).Fatal("while parsing url to nerthus", "url", uri)
		}

		reader, writer, err := websocket.Dial[message.Action](u, context.Background())
		if err != nil {
			log.WithError(err).Error("while connecting to nerthus", "url", u.String())
			time.Sleep(15 * time.Second)
			return
		}
		defer close(writer)
		num := 0
		for action := range reader {
			ActionHandler(action, writer)
			num++
			if num == 10 {
				return
			}
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
	}()
	go serv.Run()
	wg.Wait()
}

func ActionHandler(action message.Action, resp chan<- websocket.Write[message.Action]) {
	log.Info("action", "type", action.Action)
	time.Sleep(1 * time.Second)
	for i := 0; i < 5; i++ {
		resp <- websocket.Write[message.Action]{
			Data: message.Action{
				Action: action.Action,
				Response: &message.Response{
					Status:  "Finished",
					Message: fmt.Sprintf("Test message %d", i),
					Error:   nil,
				},
			},
		}
	}
	return
}
