package config

#Packages: or([ for k, v in packages {k}])
packages?: {
	#Package
}
#Package: [string]: {
	managers: [...string]
	provides: [...string]
}
