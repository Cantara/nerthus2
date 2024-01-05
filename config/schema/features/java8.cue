package config

features: java8: {
	requires: [ "zulu"]
	tasks: [
		#Install & {
			info:    "Install java 8"
			package: "zulu8-sdk"
		},
	]
}
