package config

packages: cron: {
	managers: ["apt"]
	provides: ["cron"]
}

packages: cronie: {
	managers: ["dnf", "yum"]
	provides: ["cronie"]
}
