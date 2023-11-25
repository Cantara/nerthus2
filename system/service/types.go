package service

import (
	"strconv"
	"strings"

	"github.com/cantara/bragi/sbragi"
)

type Service struct {
	Name         string       `yaml:"name"`
	ServiceType  string       `yaml:"service_type"`
	HealthType   string       `yaml:"health_type"`
	APIPath      string       `yaml:"api_path"`
	Artifact     Artifact     `yaml:"artifact"`
	Requirements Requirements `yaml:"requirements"`
}
type Requirements struct {
	RAM              string   `yaml:"ram"`
	Disk             ByteSize `yaml:"disk"`
	CPU              int      `yaml:"cpu"`
	PropertiesName   string   `yaml:"properties_name"`
	WebserverPortKey string   `yaml:"webserver_port_key"`
	NotClusterAble   bool     `yaml:"not_cluster_able"`
	IsFrontend       bool     `yaml:"is_frontend"`
	Roles            []string `yaml:"roles"`
	Services         []string `yaml:"services"`
}
type Artifact struct {
	Id       string `yaml:"id"`
	Group    string `yaml:"group"`
	Release  string `yaml:"release"`
	Snapshot string `yaml:"snapshot"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
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
