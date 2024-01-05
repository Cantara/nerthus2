package config

features: pythonAWS: {
	tasks: [
		#Install & {
			package: "boto3"
		},
		#Install & {
			package: "botocore"
		},
	]
}
