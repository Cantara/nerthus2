package config

features: zulu: {
	tasks: [
		#InstallExternal & {
			info:    "Install zulu repo"
			url:     "https://cdn.azul.com/zulu/bin/zulu-repo-1.0.0-1.noarch.rpm"
			manager: "yum"
		},
	]
	packages: {
		"zulu8-sdk": {
			managers: ["dnf", "yum"]
		}
		"zulu11-sdk": {
			managers: ["dnf", "yum"]
		}
		"zulu17-sdk": {
			managers: ["dnf", "yum"]
		}
	}
}
