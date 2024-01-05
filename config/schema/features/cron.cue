package config

features: cron: {
	custom: {
		"Amazon Linux 2023": [
			#Install & {
				info:    "Install Cron"
				package: "cronie"
			},
			#Enable & {
				info:    "Enable Cron"
				service: "crond"
			},
		]
		Debian: [
			#Install & {
				info:    "Install Cron"
				package: "cron"
			},
			#Enable & {
				info:    "Enable Cron"
				service: "cron"
			},
		]
		Ubuntu: Debian
	}
}
