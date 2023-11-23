package config

import (
	"list"
	//"io/fs"
)

_#NodeSizes: "(nano|micro|small|medium|large|xlarge|xxlarge)"

_#ClusterBase: {
	name!: string
	iam?:  string
	os:    *"Amazon Linux 2023" | "Amazon Linux 2"
	arch:  string
	size:  string
	services: [_#Service]
	services: list.MinItems(1)
	expose?: {[string]: int}
	playbook?: string
	override?: {[string]: string}
	internal:         bool | *false
	number_of_nodes?: int
	dns_root?:        string
}

_#ArmCluster: _#ClusterBase & {
	arch: "arm64"
	size: *"t4g.small" | =~#"^t4g\.\#(_#NodeSizes)$"#
}

_#x86Cluster: _#ClusterBase & {
	arch: "amd64"
	size: *"t3.small" | =~#"^t3\.\#(_#NodeSizes)$"#
}

_#Cluster: _#ArmCluster | _#x86Cluster
