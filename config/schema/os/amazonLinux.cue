package config

os: "Amazon Linux 2023": {
	packageManagers: ["dnf", "yum"]
	provides: ["usermod", "useradd", "groupadd"]
}

os: "Amazon Linux 2": {
	packageManagers: ["yum"]
}
