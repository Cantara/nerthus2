package config

#Features: or([ for k, v in features {k}])
features?: {
	[string]: {
		info?: string
		tasks?: [...#Tasks]
		custom?: [#OS]: [...#Tasks]
		requires: [...#Features]
		packages: #Package
	}
}

#Tasks: #Install | #InstallLocal | #InstallExternal | #Enable | #Download | #Link | #Delete | #FileString | #FileBytes | #Schedule | #User | #Command

_#Root:    "root"
_#System:  "system"
_#Service: "service"

#Task: {
	info?:     string
	type:      string
	privelage: string & _#Root | _#System | *_#Service
}

#Install: {
	#Task
	type:      "install"
	package:   string
	privelage: _#System
}

#InstallLocal: {
	#Task
	type:      "install_local"
	file:      string
	manager:   string
	privelage: _#System
}

#InstallExternal: {
	#Task
	type:      "install_external"
	url:       #Url
	manager:   string
	privelage: _#System
}

#Enable: {
	#Task
	type:      "enable"
	service:   string
	start:     bool | *true
	privelage: _#Root
}

#Download: {
	#Task
	type:   "download"
	source: #Url
	dest:   string
}

#Link: {
	#Task
	type:   "link"
	source: string
	dest:   string
}

#Delete: {
	#Task
	type: "delete"
	file: string
}

#FileString: {
	#Task
	type: "file_string"
	text: string
	dest: string
}

#FileBytes: {
	#Task
	type: "file_bytes"
	data: bytes
	dest: string
}

#Schedule: {
	#Task
	type:      "schedule"
	cronTime:  =~#"^(\*\/)?([1-5]?[0-9]|\*) (\*\/)?(2[0-3]|1[0-9]|[0-9]|\*) (\*\/)?(3[01]|[12][0-9]|[1-9]|\*) (\*\/)?(1[0-2]|[1-9]|\*) (\*\/)?([0-6]|\*)$"#
	command:   string
	privelage: _#System | *_#Service
}

#User: {
	#Task
	type:     "user"
	username: string
}

#Command: {
	#Task
	type: "command"
	command: [...string]
}
