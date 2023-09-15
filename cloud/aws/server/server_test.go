package server

import (
	"encoding/base64"
	"fmt"
	"testing"
)

/*
type ServerData struct {
	BuriVers  string
	CIDR      string
	CInfo     string
	CName     string
	CPorts    string
	Env       string
	NUrl      string
	Hostname  string
	IsFront   string
	NodeNames []string
	OS        string
	Arch      string
	RouteMeth string
	ServNum   string
	User      string
	VST       string
	System    string
	VUrl      string
	Webserver string
}
*/

func TestGenServerProv(t *testing.T) {
	ci := map[string]string{
		"foundation":  "foundation.foundation.nerthus.test.infra",
		"foundation2": "foundation.foundation.nerthus.test.infra",
	}
	cp := map[string]int{
		"foundation":  18030,
		"foundation2": 18032,
	}
	b, err := base64.StdEncoding.DecodeString(GenServerProv(ServerData{
		BuriVers:  "0.11.9",
		CName:     "nerthus",
		CInfo:     ci,
		CPorts:    cp,
		Env:       "test",
		NUrl:      "nerthus.test.exoreaction.dev",
		Hostname:  "test-nerthus-1",
		IsFront:   true,
		NodeNames: []string{"test-nerthus-1", "test-nerthus-2", "test-nerthus-3"},
		OS:        "linux",
		Arch:      "arm64",
		RouteMeth: "host",
		ServNum:   0,
		User:      "ec2-user",
		VST:       "H2A",
		System:    "nerthus",
		VUrl:      "visuale.test.exoreaction.dev",
		Webserver: 13030,
	}))
	fmt.Println("ProvScript: ", string(b), "\nErr: ", err)
}
