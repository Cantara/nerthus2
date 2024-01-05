package config

features: java17: {
	requires: [ "zulu"]
	tasks: [
		#Install & {
			info:    "Install java 17"
			package: "zulu17-sdk"
		},
	]
}
