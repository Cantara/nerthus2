package config

#OS: or([ for k, v in os {k}])
os?: {
	[string]: {
		packageManagers: [...#PackageManagers]
		provides: [...string]
	}
}
