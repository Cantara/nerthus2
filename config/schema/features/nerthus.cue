package config

features: nerthus: {
	requires: ["nerthus_manager", "nerthus_monitor"]
}

features: nerthus_manager: {
	tasks: [
		#Command & {
			command: ["buri", "run", "go", "-u", "-a", "nerthus2/probe", "-g", "no/cantara/gotools"]
			privelage: _#System
		},
	]
}
features: nerthus_monitor: {
	tasks: [
		#Command & {
			command: ["buri", "install", "go", "-a", "nerthus2/probe/health", "-g", "no/cantara/gotools"]
			privelage: _#Root
		},
	]
}
