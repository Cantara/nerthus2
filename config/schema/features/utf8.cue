package config

features: utf8: {
	tasks: [
		#FileString & {
			text:      "LANG=C.UTF-8\nLC_ALL=C.UTF-8"
			dest:      "/etc/environment"
			privelage: _#Root
		},
	]
}
