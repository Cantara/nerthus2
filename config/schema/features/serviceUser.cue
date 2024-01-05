package config

features: service: {
	tasks: [
		#User & {
			username:  "<service>"
			privelage: _#Root
		},
		#FileString & {
			text:      "#!/bin/sh\nsudo su - <service>"
			dest:      "./su_<service>.sh"
			privelage: _#System
		},
		#Command & {
			command: ["chmod", "+x", "./su_<service>.sh"]
			privelage: _#System
		},

	]
}
