package config

packageManagers: apt: {
	syntax: ["apt", "install", "<package>", "-y"]
	local: ["dpkg", "-i", "<file>"]
}
