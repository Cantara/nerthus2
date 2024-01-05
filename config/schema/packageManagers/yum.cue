package config

packageManagers: yum: {
	syntax: ["yum", "install", "<package>", "-y"]
	local: ["yum", "localinstall", "-y", "<file>"]
}
