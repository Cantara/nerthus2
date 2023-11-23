package config

import (
	//"io/fs"
)

_#Service: {
	name!:            string                  
	git?:            string                  
	branch?:         string                  
	webserver_port?: int              
	properties?:     string           
	dirs?:           {[string]:     string}     
	files?:          {[string]:	string}//fs.#File} 
	ServiceInfo!:    _#ServiceInfo//service.#Service 
}

_#ServiceInfo: {
	name:         string        
	service_type: string        
	health_type:  string        
	api_path:     string        
	artifact:     _#Artifact     
	requirements: _#Requirements 
}

_#Requirements: {
	ram:	 =~ #"\d+[TGMK]B"#
	disk:	 =~ #"\d+[TGMK]B"#
	cpu:                int    
	properties_name:    string 
	webserver_port_key: string 
	not_cluster_able:   bool   
	is_frontend:        bool  
	roles:		[...string] 
	services:	[..._#Service]
}

_#Artifact: {
	id:       string 
	group:    string 
	release_repo?:  string 
	snapshot_repo?: string 
	user?:     string 
	password?: string 
}
