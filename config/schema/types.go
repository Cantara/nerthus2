package schema

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cantara/bragi/sbragi"
)

type Root struct {
	OS              map[string]OS             `json:"os,omitempty"`
	Features        map[string]Feature        `json:"features,omitempty"`
	PackageManagers map[string]PackageManager `json:"packageManagers,omitempty"`
	Packages        map[string]Package        `json:"packages,omitempty"`
	Name            string                    `json:"name"`
	MachineName     string                    `json:"machine_name"`
	NerthusURL      string                    `json:"nerthus_url"`
	VisualeURL      string                    `json:"visuale_url"`
	System          System                    `json:"system"`
}
type Artifact struct {
	ID    string `json:"id"`
	Group string `json:"group"`
}
type Requirements struct {
	RAM              ByteSize `json:"ram"`
	Disk             ByteSize `json:"disk"`
	CPU              int      `json:"cpu"`
	PropertiesName   string   `json:"properties_name"`
	WebserverPortKey string   `json:"webserver_port_key"`
	NotClusterAble   bool     `json:"not_cluster_able"`
	IsFrontend       bool     `json:"is_frontend"`
	Features         []string `json:"features"`
	Packages         []string `json:"packages"`
	Services         []string `json:"services"`
}
type ServiceInfo struct {
	Name         string       `json:"name"`
	MachineName  string       `json:"machine_name"`
	ServiceType  string       `json:"service_type"`
	HealthType   string       `json:"health_type"`
	APIPath      string       `json:"api_path"`
	Artifact     Artifact     `json:"artifact"`
	Requirements Requirements `json:"requirements"`
}
type Service struct {
	Name        string      `json:"name"`
	MachineName string      `json:"machine_name"`
	Props       string      `json:"props"`
	Port        int         `json:"port"`
	Definition  ServiceInfo `json:"definition"`
}
type Cluster struct {
	Name        string    `json:"name"`
	MachineName string    `json:"machine_name"`
	Node        Node      `json:"node"`
	Size        int       `json:"size"`
	Services    []Service `json:"services"`
	Internal    bool      `json:"internal"`
}
type Node struct {
	Os   string `json:"os"`
	Arch Arch   `json:"arch"`
	Size string `json:"size"`
}
type RoutingMethod string

const (
	HostRouting = RoutingMethod("host")
	PathRouting = RoutingMethod("path")
)

type System struct {
	Name          string        `json:"name"`
	MachineName   string        `json:"machine_name"`
	Domain        string        `json:"domain"`
	RoutingMethod RoutingMethod `json:"routing_method"`
	Cidr          string        `json:"cidr"`
	Zone          string        `json:"zone"`
	Clusters      []Cluster     `json:"clusters"`
}
type Task struct {
	Info      string   `json:"info,omitempty"`
	Type      string   `json:"type,omitempty"`
	Source    string   `json:"source,omitempty"`
	Dest      string   `json:"dest,omitempty"`
	File      string   `json:"file,omitempty"`
	Url       string   `json:"url,omitempty"`
	Manager   string   `json:"manager,omitempty"`
	Package   string   `json:"package,omitempty"`
	Service   string   `json:"service,omitempty"`
	Start     bool     `json:"start,omitempty"`
	Text      string   `json:"text,omitempty"`
	Data      []byte   `json:"data,omitempty"`
	Command   []string `json:"command"`
	Username  string   `json:"username,omitempty"`
	Privelage string   `json:"privelage"`
}
type Feature struct {
	Friendly string   `json:"friendly,omitempty"`
	Requires []string `json:"requires,omitempty"`
	Tasks    []Task   `json:"tasks,omitempty"`
	Custom   map[string][]Task
	Packages map[string]Package `json:"packages,omitempty"`
}

func (f Feature) Service(os string) bool {
	tasks, ok := f.Custom[os]
	if !ok {
		tasks = f.Tasks
	}
	for _, t := range tasks {
		sbragi.Info("privelaged", "type", t.Type, "privelage", t.Privelage, "install?", strings.HasPrefix(t.Type, "install"), "enable?", t.Type == "enable")
		if t.Privelage != "service" {
			return false
		}
	}
	return true
}

type PackageManager struct {
	Syntax   []string `json:"syntax,omitempty"`
	Local    []string `json:"local,omitempty"`
	Requires []string `json:"requires,omitempty"`
	Root     bool     `json:"root"`
}
type Package struct {
	Managers []string `json:"managers,omitempty"`
	Provides []string `json:"provides,omitempty"`
}
type OS struct {
	PackageManagers []string `json:"packageManagers,omitempty"`
	Provides        []string `json:"provides,omitempty"`
}
type Arch string

const (
	Arm64  Arch = "arm64"
	X86_64 Arch = "x86_64"
)

func (a Arch) String() string {
	switch a {
	case X86_64:
		fallthrough
	case Arm64:
		return string(a)
	}
	return "INVALID ARCH"
}

func (a Arch) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a *Arch) UnmarshalText(b []byte) error {
	var err error
	*a, err = StringToArch(string(b))
	return err
}

func StringToArch(s string) (Arch, error) {
	switch strings.ToUpper(s) {
	case "AMD64":
		fallthrough
	case "X86-64":
		fallthrough
	case "X86_64":
		return X86_64, nil
	case "ARM64":
		return Arm64, nil
	}
	return "", fmt.Errorf("%s is not a valid arch", s)
}

type ByteSize string

const (
	_ = 1 << (10 * iota)
	KB
	MB
	GB
	TB
	PB
)

func (size ByteSize) ToGB() int {
	switch strings.ToUpper(string(size[len(size)-2:])) {
	case "KB":
		size, err := strconv.Atoi(string(size[:len(size)-2]))
		if err != nil {
			sbragi.WithError(err).Error("while getting KB size")
			return 0
		}
		return int((float64(size*KB) + (0.5 * GB)) / GB)
	case "MB":
		size, err := strconv.Atoi(string(size[:len(size)-2]))
		if err != nil {
			sbragi.WithError(err).Error("while getting MB size")
			return 0
		}
		return int((float64(size*MB) + (0.5 * GB)) / GB)
	case "GB":
		size, err := strconv.Atoi(string(size[:len(size)-2]))
		if err != nil {
			sbragi.WithError(err).Error("while getting GB size")
			return 0
		}
		return size
	case "TB":
		size, err := strconv.Atoi(string(size[:len(size)-2]))
		if err != nil {
			sbragi.WithError(err).Error("while getting TB size")
			return 0
		}
		return size * TB / GB
	case "PB":
		size, err := strconv.Atoi(string(size[:len(size)-2]))
		if err != nil {
			sbragi.WithError(err).Error("while getting PB size")
			return 0
		}
		return size * PB / GB
	}
	sbragi.Error("should not be hit", "size", size[len(size)-2:])
	return 0
}
