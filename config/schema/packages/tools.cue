package config

packages: htop: {
	managers: ["apt", "dnf", "yum"]
	provides: ["htop"]
}

packages: jq: {
	managers: ["apt", "dnf", "yum"]
	provides: ["jq"]
}
