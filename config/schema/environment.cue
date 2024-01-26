package config

import "strings"

#Uri: =~#"^[^:/\.\s][^:/\s]*\.[^:/\.\s]+$"#
#Url: =~#"^(https:\/\/)?([^:/\.\s][^:/\s]*\.)+[^:\/\.\s]+(\/[^:\/\s]+)*?$"#

name:         string
machine_name: strings.Replace(name, " ", "-", -1)
nerthus_url:  #Uri
visuale_url:  #Uri
system:       #System
