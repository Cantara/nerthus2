package config

packages: docker: {
	managers: ["apt", "dnf", "yum"]
	provides: ["docker"]
}
