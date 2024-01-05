package config

features: buri: {
	#version: "0.12.5"
	#os:      "linux"
	#arch:    "amd64"
	#name:    "buri-v\(#version)-\(#os)-\(#arch)"
	#folder:  "/usr/local/bin"
	#dest:    "\(#folder)/\(#name)"
	tasks: [
		#Download & {
			source:    "https://mvnrepo.cantara.no/content/repositories/releases/no/cantara/gotools/buri/v\(#version)/\(#name)"
			dest:      #dest
			privelage: _#Root
		},
		#Command & {
			command: ["chmod", "+x", #dest]
			privelage: _#Root
		},
		#Link & {
			source:    "\(#folder)/buri"
			dest:      #dest
			privelage: _#Root
		},
	]
}
