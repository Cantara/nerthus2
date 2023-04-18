# Nerthus

## Getting started (as user of existing nerthus provisioned environment)

* Log on to https://nerthus.lab.cantara.no using the GitHub authentication method
* Look at the current installation in the UI
* To get synced access to nodes
  * Press the generate&download button to create and downlod the access-sync agent, and install this on the home-directory you want to have synced-access scripts for the nodes
  * Note:  If someone trigger the "Break the glass"-action, all users will have to log-in and re-create their access-sync agent.
* 

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
