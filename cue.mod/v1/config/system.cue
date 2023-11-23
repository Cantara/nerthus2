package config

import (
	"list"
	//"io/fs"
)

_#IPRegex: #"(\b25[0-5]|\b2[0-4][0-9]|\b[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}"#

_#RoutingMethod: _#enumRoutingMethod

_#enumRoutingMethod:
	_#RoutingPath |
	_#RoutingHost

_#RoutingPath: _#RoutingMethod & "path"
_#RoutingHost: _#RoutingMethod & "host"

_#System: {
	name:           string
	domain!:        _#Url //=~ #"[^\.\s]\S+\.\S+[^\.\s]"# //Think i can use string interpolation here to use the url definition without the https part, aka split then in to two //Would like to use a domain in environment to define the end of this domain. or change it to not include tld here, aka just the subdomain part and make it optional
	routing_method: _#RoutingMethod
	cidr:           =~"^\(_#IPRegex)" & =~#"\.0\/24$"#
	zone:           string
	clusters:       [..._#Cluster] & list.MinItems(1)
}
