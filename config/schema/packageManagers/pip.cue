package config

packageManagers: pip: {
	syntax: ["sh", "-c", "yes | pip install <package>"]
	root: false
	requires: ["python"]
}
