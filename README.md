# Nerthus

## User client

Nerthus provides a cli tool to help you work with services at the node level.

### Current gotchas

There is a issue with the WebSocket client-server solution, so if an action does not execute propperly, try repeating it.
This should be a very sparse issue, but can be especially precent when there has been a long time since any actions have been done.

### Giude

#### Install the eXOReaction nexus package manager and nerthus-cli for all users

``` bash
sudo curl https://github.com/cantara/nerthus2/install_cli.sh -sSf | sh
```

#### Add crontab entries for [brui](https://github.com/cantara/buri) and nerthus-cli

This currently requires `sudo` without password to update packages. There is currently no support for local user insallations. Look at [brui](https://github.com/cantara/buri) for the most up to date information for the most up to date information.

``` bash
curl https://github.com/cantara/nerthus2/add_cron.sh -sSf | sh
```

#### Add config (Remove me)

This is a temporary solution until github loggin is added to nerthus.

``` bash
touch ~/.config/.nerthus-cli.yaml
```

## Environment configurations

The information here is not an exhaustive list of all options or modifications that are possible, but it will try to give a short and simple overview of the most important options that are available.

### Filestructure

#### Folders

 * ansible
    Contains ansible scripts used by the nerthus service
 * [roles](#Roles)
    Contains ansible scripts that are used for building scripts that are sent to the nerthus probes
 * services
    Contains local definitions of services
 * systems
    Contains configurations for the different tightly coupled systems your environment is built up of

#### Files

 * config.yml
    Holds envionment spesific and top level configurations for the environment

### config.yml

 * name: prod
 * domain: "greps.dev"
 * os_name: Amazon Linux 2023
 * os_arch: arm64
 * instance_type: t4g.micro
 * visuale_host: visuale.greps.dev
 * nerthus_host: nerthus.greps.dev
 * vars:
 * public_domain: greps.dev
 * systems:
    #- jenkins
    - nerthus
    - visuale
    #- nexus

### Ansible

The ansible folder contains two files. These define the provisioning scripts. By default these will handle setting up the AWS envionment with all that follows.

 * provision.yml
 * loadbalancer.yml

### Roles

The roles folder contains zero or more files that contain minimal ansible "roles". These are used to define the desired state on a node.
Think of these as standalone playbooks that are chained together with a shared pool of vars.

### Services

The services folder contains zero or more local service declerations. [Example](https://github.com/cantara/nerthus2/nerthus.yml)

 * name: <string> (Used to identify the service on the nodes)
 * service_type: <[ServiceTypes](https://github.com/cantara/visuale)>
 * health_type: <[HealthTypes](#Health_types)
 * artifact:
    * id: <string> (maven artifact id)
    * group: <string> (maven artifact group id)
    * release: <string> (url to nexus release repo)
    * snapshot: <string> (url to nexus snapshot repo)
    * user: <string> (username used for nexus repo auth)
    * password: <string> (password used for nexus repo auth)
 * requirements:
    * ram: <size> (WIP)
    * disk: <size> (WIP)
    * cpu: <uint> (WIP)
    * properties_name: <string> (Should deprecate?)
    * webserver_port_key: <string> (Should deprecate?)
    * not_cluster_able: <bool> "DEFAULT: true" (Defining weather or not this service can be run in a cluster)
    * is_frontend: <bool> "DEFAULT: false" (Defines if the service requires loadbalancer routing on base route)
    * roles: <list <string>> (Defines the roles this service requires)
    * services: <list <string>> (Defines the services this service requires tight coupling to)

### Systems

The systems folder contains folders for each tightly coupled system.

#### System configurations

Configurations and information related to a tightly coupled system

##### Folders

 * files
    Contains files that is getting added to services on nodes.
 * [roles](#Roles)
    Contains roles that are only used by the spesific system.
 * [services](#Services)

##### Files

 * config.yml
    Contains configuration for all clusters and services in one tightly coupled system

###### config.yml

 * name: <string> (Used for identifying this system)
 * cidr_base: <ip> "Ex: 10.1.255" (Used to create /24 subnets with the given prefix)
 * routing_method: <[RoutingMethod](#Routing_methods)> (Used to define strategy in the loadbalancer and for health reporting)
 * os_name: <[OSName](#OS_name)>
 * os_arch: <[Arch](#System_architecture)>
 * instance_type: <[InstanceSize](https://aws.amazon.com/ec2/instance-types/)>
 * vars: <map[string]string> (variable map with same rules as [Ansible](https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_variables.html), these will be used by roles on nodes)
 * clusters: <list<cluster>>
    * - name: <string> (Used for identifying this cluster)
        * os_name: <[OSName](#OS_name)>
        * os_arch: <[Arch](#System_architecture)>
        * instance_type: <[InstanceSize](https://aws.amazon.com/ec2/instance-types/)>
        * services:
            * - name: <string> (Used for identifying this service)
                * local: <filename> (Local service decleration file)
                * git: <github_repo> "Ex: github.com/Cantara/nerthus2" (Github repo with service decleration file)
                * branch: <string> (Branch to find nerhus.yml service decleration)
                * webserver_port: <uint> (Used to expose service to loadbalancer)
                * dirs:
                    * \<relative path within files>: <path from service dir>
                * files:
                    * \<path from service dir>:
                        * mode:  <unix permission> "Ex: 0640"
                        * content: <string>
        * override: <map[string]string>
        * expose: <map[string]uint> (Ports that gets exposed to services that requires this service)

## Future Getting started (as user of existing nerthus provisioned environment)

* [ ] Log on to https://nerthus.lab.cantara.no using the GitHub authentication method
* [ ] Look at the current installation in the UI
* [ ] To get synced access to nodes
  * [ ] Press the generate&download button to create and downlod the access-sync agent, and install this on the home-directory you want to have synced-access scripts for the nodes
  * [ ] Note:  If someone trigger the "Break the glass"-action, all users will have to log-in and re-create their access-sync agent.
* [ ] To add a new service

## Setting up a new nerthus provisioned environment

## Vars

### Replacements

* All `-` will be replaced with `_`, so use `_` in playbooks

### Provisioning
    region: ap-northeast-1
    #os_name: Ubuntu Kinetic 22.10
    os_name: Amazon Linux 2023
    #os_name: Amazon Linux 2
    #os_name: Debian 11
    os_arch: arm64
    instance_type: t4g.nano #t3.small
    cidr_base: 10.100.0
    service: nerthus
    system: cantara-lab
    zone: lab.cantara.infra
    key_name:
    vpc_name:
    security_group_name:
    node_names: []
    target_group_name:
    loadbalancer_name:
    security_group_rules: []
    name_base:
    iam_profile:
    bootstrap:
    webserver_port:
    is_frontend: false
