package start

import (
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/story"
)

type Start struct {
	Env      string   `json:"env"`
	System   string   `json:"system"`
	Cluster  string   `json:"cluster"`
	OSName   string   `json:"os_name"`
	Arch     ami.Arch `json:"arch"`
	Network  string   `json:"network"`
	Nodes    []string `json:"node_names"`
	Size     string   `json:"size"`
	DiskSize int      `json:"disk_size"`
	Nerthus  string   `json:"nerthus"`
	Visuale  string   `json:"visuale"`
	Path     string   `json:"path"`
	Port     int      `json:"port"`
	Base     string   `json:"base"`
}

var Fingerprint = adapter.New[Start](story.AdapterStart)
var Adapter = Fingerprint.Adapter(func(a []adapter.Value) (s Start, err error) {
	return Fingerprint.Value(a[0]), nil
}, Fingerprint)
