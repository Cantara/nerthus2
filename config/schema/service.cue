package config

import "strings"

#Service: {
	name:         string
	machine_name: strings.Replace(name, " ", "-", -1)
	git?:         string
	branch?:      string
	port?:        int
	props?:       string
	dirs?: {[string]: string}
	files?: {[string]: string}
	definition: #ServiceDefinition
}

#ServiceDefinition: {
	name:         string
	machine_name: strings.Replace(name, " ", "-", -1)
	service_type: string
	health_type:  string
	api_path?:    string
	artifact?:    #Artifact
	docker?:      #Docker
	requirements: #Requirements
}

#Requirements: {
	ram:                 =~#"\d+[TGMK]B"#
	disk:                =~#"\d+[TGMK]B"#
	cpu:                 int
	properties_name:     string
	webserver_port_key?: string
	not_cluster_able:    bool | *false
	is_frontend:         bool | *false
	features: [...#Features]
	packages: [...#Packages]
	services: [...string]
}

#Artifact: {
	id:             string
	group:          string
	release_repo?:  string
	snapshot_repo?: string
	user?:          string
	password?:      string
}

#Docker: {
	image:    string
	tag:      string
	x86Tag:   string | *tag
	arm64Tag: string | *tag
	ports: [...int]
	volumes: [...string]
	env: [string]: string
	restart: bool | *true
	user:    string | *"<user>"
	group:   string | *"<group>"
}
