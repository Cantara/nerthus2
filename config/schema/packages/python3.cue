package config

packages: python3: {
	managers: ["apt", "dnf", "yum"]
	provides: ["python3"]
}

packages: "python3-pip": {
	managers: ["apt", "dnf", "yum"]
	provides: ["pip3"]
}
