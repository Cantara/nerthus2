package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	log "github.com/cantara/bragi"
	"os"
	"path/filepath"

	//jsoniter "github.com/json-iterator/go"
	"go/types"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//var json = jsoniter.ConfigCompatibleWithStandardLibrary

var duration time.Duration
var interval time.Duration
var reportURLString string
var healthURLString string
var serviceTypeSelectedString string
var artifactID string

func init() {
	const (
		defaultDuration    = time.Minute
		durationUsage      = "duration to run"
		defaultInterval    = time.Second * 5
		intervalUsage      = "interval between health checks"
		defaultReportURL   = ""
		reportURLUsage     = "url to report health to ex: https://visuale.cantara.no/api/status/ENV/NAME/host_undefined?service_tag=undefined&service_type=A2A"
		defaultHealthURL   = "http://localhost:3030/health"
		healthURLUsage     = "url to get health from"
		defaultServiceType = string(defaultST)
		serviceTypeUsage   = "type of service to probe. default, java, jar, go, eventstore, es"
		defaultArtifactID  = ""
		artifactIDUsage    = "artifact to probe health from"
	)
	flag.DurationVar(&duration, "duration", defaultDuration, durationUsage)
	flag.DurationVar(&duration, "d", defaultDuration, durationUsage+" (shorthand)")
	flag.DurationVar(&interval, "interval", defaultInterval, intervalUsage)
	flag.DurationVar(&interval, "i", defaultInterval, intervalUsage+" (shorthand)")
	flag.StringVar(&reportURLString, "report-url", defaultReportURL, reportURLUsage)
	flag.StringVar(&reportURLString, "r", defaultReportURL, reportURLUsage+" (shorthand)")
	flag.StringVar(&healthURLString, "health-url", defaultHealthURL, healthURLUsage)
	flag.StringVar(&healthURLString, "h", defaultHealthURL, healthURLUsage+" (shorthand)")
	flag.StringVar(&serviceTypeSelectedString, "service-type", defaultServiceType, serviceTypeUsage)
	flag.StringVar(&serviceTypeSelectedString, "t", defaultServiceType, serviceTypeUsage+" (shorthand)")
	flag.StringVar(&artifactID, "artifact-id", defaultArtifactID, artifactIDUsage)
	flag.StringVar(&artifactID, "a", defaultArtifactID, artifactIDUsage+" (shorthand)")
}

var version string

func main() {
	flag.Parse()
	reportURL, err := url.ParseRequestURI(reportURLString)
	if err != nil {
		log.AddError(err).Fatal("report url has to be a valid url")
		return
	}
	healthURL, err := url.ParseRequestURI(healthURLString)
	if err != nil {
		log.AddError(err).Fatal("report url has to be a valid url")
		return
	}
	serviceTypeSelected, err := serviceTypeFromString(serviceTypeSelectedString)
	if err != nil {
		log.AddError(err).Fatal("service type has to be a valid service type")
		return
	}
	endTime := time.Now().Add(duration)
	t := time.NewTicker(interval)
	switch serviceTypeSelected {
	case eventstoreST:
		version = "22.10"
	case javaST:
		version, err = versionFromLink(".jar")
		if err != nil {
			log.AddError(err).Fatal("while getting version of artifact")
			return
		}
	case goST:
		version, err = versionFromLink("")
		if err != nil {
			log.AddError(err).Fatal("while getting version of artifact")
			return
		}
	}
	for endTime.After(time.Now()) {
		select {
		case <-t.C:
			var status any
			switch serviceTypeSelected {
			case eventstoreST:
				status, err = EventStoreStatus(healthURL)
			default:
				status, err = DefaultStatus(healthURL)
			}
			if err != nil {
				log.AddError(err).Error("while reading status")
				err = Put[baseStatus, types.Nil](reportURL, &baseStatus{
					Status:  "FAIL",
					Name:    "",
					Version: version,
					IP:      GetOutboundIP(),
					Now:     time.Now(),
				}, nil)
				if err != nil {
					log.AddError(err).Crit("while posting status")
					continue
				}
				continue
			}
			err = Put[any, types.Nil](reportURL, &status, nil)
			if err != nil {
				log.AddError(err).Crit("while posting status")
				continue
			}
			fmt.Println(status)
		}
	}
}

func EventStoreStatus(healthURL *url.URL) (out any, err error) {
	healthURL.Path = "/stats"
	var status map[string]interface{}
	err = Get(healthURL, &status)
	if err != nil {
		return
	}
	var since time.Time
	since, err = time.Parse("2006-01-02T15:04:05Z", status["proc"].(map[string]interface{})["startTime"].(string))
	if err != nil {
		return
	}
	healthURL.Path = "/gossip"
	var goss gossip
	err = Get(healthURL, &goss)
	if err != nil {
		return
	}
	out = eventstoreStatus{
		baseStatus: baseStatus{
			Status:       "UP",
			Name:         "eventstore",
			Version:      version,
			IP:           GetOutboundIP(),
			Now:          time.Now(),
			RunningSince: &since,
		},
		NodesInCluster: uint(len(goss.Members)),
		Gossip:         goss,
	}
	return
}

func DefaultStatus(healthURL *url.URL) (out any, err error) {
	err = Get(healthURL, &out)
	if err != nil {
		return
	}
	return
}

func Put[I, O any](uri *url.URL, data *I, out *O) (err error) {
	jsonValue, err := json.Marshal(data)
	if err != nil {
		return
	}
	log.Println(string(jsonValue))
	client := &http.Client{}
	log.Println("Posting health to: ", uri.String())
	req, err := http.NewRequest("PUT", uri.String(), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	fmt.Println(resp)
	if err != nil || out == nil {
		return err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.AddError(err).Error("while closing response body")
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, out)
	if err != nil {
		log.AddError(err).Warning(fmt.Sprintf("%s\t%v", body, data))
	}

	return
}

func Get[O any](uri *url.URL, out *O) (err error) {
	client := &http.Client{}
	log.Println("Getting health at: ", uri.String())
	req, err := http.NewRequest("GET", uri.String(), nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil || out == nil {
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.AddError(err).Error("while closing response body")
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, out)
	if err != nil {
		log.AddError(err).Warning(body)
	}
	return
}

type baseStatus struct {
	Status       string     `json:"status"`
	Name         string     `json:"name"`
	Version      string     `json:"version"`
	IP           net.IP     `json:"ip"`
	Now          time.Time  `json:"now"`
	RunningSince *time.Time `json:"running_since,omitempty"`
}

type eventstoreStatus struct {
	baseStatus

	NodesInCluster uint   `json:"nodesInCluster"`
	Gossip         gossip `json:"gossip"`
}

var ip net.IP

func GetOutboundIP() net.IP {
	if ip != nil {
		return ip
	}
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip = localAddr.IP

	return ip
}

const (
	defaultST    = serviceType("default")
	javaST       = serviceType("java")
	goST         = serviceType("go")
	eventstoreST = serviceType("eventstore")
)

type serviceType string

func (s *serviceType) String() string {
	return fmt.Sprint(*s)
}

func serviceTypeFromString(s string) (st serviceType, err error) {
	switch strings.ToLower(s) {
	case "java":
		st = javaST
	case "jar":
		st = javaST
	case "go":
		st = goST
	case "eventstore":
		st = eventstoreST
	case "es":
		st = eventstoreST
	default:
		err = errors.New("unsuported service type")
	}
	return
}

type gossip struct {
	Members []struct {
		InstanceId        string    `json:"instanceId"`
		TimeStamp         time.Time `json:"timeStamp"`
		State             string    `json:"state"`
		IsAlive           bool      `json:"isAlive"`
		HttpEndPointIp    string    `json:"httpEndPointIp"`
		HttpEndPointPort  int       `json:"httpEndPointPort"`
		IsReadOnlyReplica bool      `json:"isReadOnlyReplica"`
	} `json:"members"`
	ServerIp   string `json:"serverIp"`
	ServerPort int    `json:"serverPort"`
}

func versionFromLink(ext string) (ver string, err error) {
	link, err := os.Readlink(artifactID + ext)
	if err != nil {
		return
	}
	name := filepath.Base(link)
	ver = strings.ReplaceAll(strings.ReplaceAll(name, ext, ""), artifactID+"-", "")
	return
}
