package config

import "list"

import "strings"

#NodeSizes: "(nano|micro|small|medium|large|xlarge|xxlarge)"

//_#clusterBase: {
#Cluster: {
	name:         string
	machine_name: strings.Replace(name, " ", "-", -1)
	iam?:         string
	node:         #ArmNode | #x86Node
	size:         >0 & int | *3
	services: [...#Service] & list.MinItems(1)
	expose?: {[string]: int}
	playbook?: string
	override?: {[string]: string}
	internal:         bool | *false
	number_of_nodes?: int
	dns_root?:        string
}

_#node: {
	os:   #OS | *"Amazon Linux 2023"
	arch: string
	size: string
}

#ArmNode: _#node & {
	arch: "arm64"
	size: *"t4g.small" | =~#"^t4g\.\#(#NodeSizes)$"#
}

#x86Node: _#node & {
	arch: "amd64"
	size: *"t3.small" | =~#"^t3\.\#(#NodeSizes)$"#
}
