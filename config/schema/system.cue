package config

import "list"

import "strings"

#IPRegex: #"(\b25[0-5]|\b2[0-4][0-9]|\b[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}"#

#RoutingMethod: #enumRoutingMethod

#enumRoutingMethod:
	#RoutingPath |
	#RoutingHost

#RoutingPath: #RoutingMethod & "path"
#RoutingHost: #RoutingMethod & "host"

#System: {
	name:           string
	machine_name:   strings.Replace(name, " ", "-", -1)
	domain:         #Url
	routing_method: #RoutingMethod
	cidr:           =~"^\(#IPRegex)" & =~#"\.0\/24$"#
	zone:           string
	clusters: [...#Cluster] & list.MinItems(1)
}
