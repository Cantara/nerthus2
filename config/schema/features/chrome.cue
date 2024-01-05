package config

features: chrome: {
	custom: {
		_#ChromeRPM: "/tmp/google-chrome-stable_current_x86_64.rpm"
		"Amazon Linux 2023": [
			#Download & {
				info:   "Download package"
				source: "https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm"
				dest:   _#ChromeRPM
			},
			#InstallLocal & {
				info:    "Import and install package"
				file:    _#ChromeRPM
				manager: "yum"
			},
			#Delete & {
				info: "Delete local package file"
				file: _#ChromeRPM
			},
		]
		_#ChromeDEB: "/tmp/google-chrome-stable_current_amd64.deb"
		debian: [
			#Download & {
				info:   "Download package"
				source: "https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb"
				dest:   _#ChromeDEB
			},
			#InstallLocal & {
				info:    "Import and install package"
				file:    _#ChromeDEB
				manager: "apt"
			},
			#Delete & {
				info: "Delete local package file"
				file: _#ChromeDEB
			},
		]
	}
}
