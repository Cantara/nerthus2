package config

features: java11: {
	requires: [ "zulu"]
	tasks: [
		#Install & {
			info:    "Install java 11"
			package: "zulu11-sdk"
		},
	]
}
