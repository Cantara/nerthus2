package config

import "list"

//_#Url: =~ #"^(?:https:\/\/)?[^:/\.\s][^:/\s]*\.[^:/\.\s]+$"#
_#Url: =~#"^[^:/\.\s][^:/\s]*\.[^:/\.\s]+$"#

_#Environment: {
	name!:        string
	nerthus_url!: _#Url
	visuale_url!: _#Url
	systems:      [..._#System] & list.MinItems(1)
}
