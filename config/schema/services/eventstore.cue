package config

_#EventStoreSD: #ServiceDefinition & {
	name:         "eventstore"
	service_type: "CS"
	health_type:  "eventstore"
	docker: {
		image:    "eventstore/eventstore"
		tag:      "latest"
		x86Tag:   "latest"
		arm64Tag: "22.10.1-alpha-arm64v8"
		ports: [2113, 1112]
		volumes: [
			"/var/lib/eventstore",
			"/var/log/eventstore",
		]
		env: {
			EVENTSTORE_INSECURE:                   "true"
			EVENTSTORE_EXT_IP:                     "0.0.0.0"
			EVENTSTORE_EXT_HOST_ADVERTISE_AS:      "<ip>"
			EVENTSTORE_INT_IP:                     "0.0.0.0"
			EVENTSTORE_INT_HOST_ADVERTISE_AS:      "<ip>"
			EVENTSTORE_CLUSTER_SIZE:               "<num_nodes>"
			EVENTSTORE_CLUSTER_DNS:                "<cluster_name>"
			EVENTSTORE_MAX_APPEND_SIZE:            "8388608"
			EVENTSTORE_ENABLE_ATOM_PUB_OVER_HTTP:  "false"
			EVENTSTORE_RUN_PROJECTIONS:            "" //TODO: FIXME: missing value
			EVENTSTORE_START_STANDARD_PROJECTIONS: "true"
		}
	}
	requirements: {
		ram:             "2GB"
		disk:            "30GB"
		cpu:             2
		properties_name: "docker"
		features: [
			"docker",
			"buri",
			"cron",
		]
		packages: []
		services: []
	}
}
