package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/key"
	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/vpc"

	log "github.com/cantara/bragi/sbragi"
)

type Server struct {
	Name               string
	Cluster            string
	Id                 string
	Type               string
	PublicDNS          string
	VolumeId           string `json:"volume_id"`
	NetworkInterfaceId string `json:"network_interface_id"`
	ami                ami.Image
	key                key.Key
	group              security.Group
}

func NewServer(name, cluster string, image ami.Image, key key.Key, group security.Group, e2 *ec2.Client) (s Server, err error) {
	return
}

func GetServer(name string, e2 *ec2.Client) (s Server, err error) {
	result, err := e2.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []string{
					name,
				},
			},
		},
	})
	if err != nil {
		return
	}
	if len(result.Reservations) < 1 {
		err = fmt.Errorf("error: %w name=%s", ErrServerNotFound, name)
		return
	}
	/* if len(result.Reservations) > 1 {
		err = fmt.Errorf("Too many servers with name %s", name)
	} */
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State.Name != ec2types.InstanceStateNameRunning && instance.State.Name != ec2types.InstanceStateNamePending {
				continue
			}

			if len(instance.BlockDeviceMappings) < 1 || len(instance.NetworkInterfaces) < 1 {
				continue
			}
			s = Server{
				Name:               name,
				Cluster:            vpc.Tag(instance.Tags, "Cluster"),
				Id:                 aws.ToString(instance.InstanceId),
				PublicDNS:          aws.ToString(instance.PublicDnsName),
				VolumeId:           aws.ToString(instance.BlockDeviceMappings[0].Ebs.VolumeId),
				NetworkInterfaceId: aws.ToString(instance.NetworkInterfaces[0].NetworkInterfaceId),
			}
			return
		}
	}
	err = fmt.Errorf("error: %w, name=%s", ErrServerNotFound, name)
	return
}

func NameAvailable(name string, e2 *ec2.Client) (available bool, err error) {
	result, err := e2.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []string{
					name,
				},
			},
		},
	})
	if err != nil {
		return
	}
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State.Name == ec2types.InstanceStateNameTerminated {
				continue
			}
			available = false
			return
		}
	}
	available = true
	return
}

func Create(nodeNum int, node, cluster, system, env, iType, subnet, nerthusUrl, visualeUrl string, image ami.Image, key key.Key, group security.Group, e2 *ec2.Client) (Server, error) {
	s, err := GetServer(node, e2)
	if err != nil {
		if !errors.Is(err, ErrServerNotFound) {
			return Server{}, err
		}
		err = nil
	} else {
		log.Trace("Server already exists", "name", node)
		return s, nil
	}
	// Specify the details of the instance that you want to create
	s = Server{
		Name:    node,
		Cluster: cluster,
		ami:     image,
		key:     key,
		group:   group,
		Type:    iType,
	}
	ProvScript := GenServerProv(ServerData{
		BuriVers: "0.11.9",
		CName:    cluster,
		Env:      env,
		NUrl:     nerthusUrl,
		Hostname: node,
		OS:       "linux",
		Arch:     image.Arch.String(),
		ServNum:  nodeNum,
		User:     image.Username(),
		System:   system,
		VUrl:     visualeUrl,
	})
	result, err := e2.RunInstances(context.Background(), &ec2.RunInstancesInput{
		ImageId:      &s.ami.Id,
		InstanceType: ec2types.InstanceType(s.Type), //"t3.micro",
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		KeyName:      aws.String(s.key.Name),
		UserData:     &ProvScript,
		NetworkInterfaces: []ec2types.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: aws.Bool(true),
				DeleteOnTermination:      aws.Bool(true),
				DeviceIndex:              aws.Int32(0),
				Groups: []string{
					group.Id,
				},
				Ipv6AddressCount: aws.Int32(1),
				PrimaryIpv6:      aws.Bool(true),
				SubnetId:         &subnet,
			},
		},
		BlockDeviceMappings: []ec2types.BlockDeviceMapping{
			{
				DeviceName: &s.ami.RootDev,
				Ebs: &ec2types.EbsBlockDevice{
					VolumeSize: aws.Int32(20),
					VolumeType: "gp3",
				},
			},
		},
		MetadataOptions: &ec2types.InstanceMetadataOptionsRequest{
			HttpTokens: ec2types.HttpTokensStateRequired,
		},
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(s.Name),
					},
					{
						Key:   aws.String("Cluster"),
						Value: aws.String(s.Cluster),
					},
					{
						Key:   aws.String("Manager"),
						Value: aws.String("nerthus"),
					},
					{
						Key:   aws.String("OS"),
						Value: aws.String(image.HName),
					},
					{
						Key:   aws.String("Arch"),
						Value: aws.String(image.Arch.String()),
					},
				},
			},
			{
				ResourceType: ec2types.ResourceTypeVolume,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(s.Name),
					},
					{
						Key:   aws.String("Cluster"),
						Value: aws.String(s.Cluster),
					},
					{
						Key:   aws.String("OS"),
						Value: aws.String(image.HName),
					},
					{
						Key:   aws.String("Arch"),
						Value: aws.String(image.Arch.String()),
					},
				},
			},
		},
	})
	if err != nil {
		err = fmt.Errorf("Could not create instance with name %s. err: %v", s.Name, err)
		return Server{}, err
	}
	s.Id = aws.ToString(result.Instances[0].InstanceId)
	s.NetworkInterfaceId = aws.ToString(result.Instances[0].NetworkInterfaces[0].NetworkInterfaceId)
	//s.VolumeId = aws.ToString(result.Instances[0].BlockDeviceMappings[0].Ebs.VolumeId)
	return s, nil
}

func (s *Server) Delete(e2 *ec2.Client) (err error) {
	_, err = e2.TerminateInstances(context.Background(), &ec2.TerminateInstancesInput{
		InstanceIds: []string{
			s.Id,
		},
	})
	if err != nil {
		return
	}
	err = s.WaitUntilTerminated(e2)
	return
}

func WaitUntilRunning(ids []string, e2 *ec2.Client) (err error) {
	err = ec2.NewInstanceRunningWaiter(e2).Wait(context.Background(), &ec2.DescribeInstancesInput{
		InstanceIds: ids,
	}, 5*time.Minute)
	if err != nil {
		return
	}
	return
}

func (s Server) WaitUntilTerminated(e2 *ec2.Client) (err error) {
	err = ec2.NewInstanceTerminatedWaiter(e2).Wait(context.Background(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{s.Id},
	}, 5*time.Minute)
	return
}

func (s Server) WaitUntilNetworkAvailable(e2 *ec2.Client) (err error) {
	err = ec2.NewNetworkInterfaceAvailableWaiter(e2).Wait(context.Background(), &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{s.NetworkInterfaceId},
	}, 5*time.Minute)
	return
}

func (s *Server) GetPublicDNS(e2 *ec2.Client) (publicDNS string, err error) {
	if s.PublicDNS != "" {
		publicDNS = s.PublicDNS
		return
	}
	result, err := e2.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{s.Id},
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe Instance. err: %v", err)
		return
	}
	s.PublicDNS = aws.ToString(result.Reservations[0].Instances[0].PublicDnsName)
	publicDNS = s.PublicDNS
	return
}

func (s *Server) GetVolumeId(e2 *ec2.Client) (volumeId string, err error) {
	if s.VolumeId != "" {
		volumeId = s.VolumeId
		return
	}
	result, err := e2.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{s.Id},
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe Instance. err: %v", err)
		return
	}
	s.VolumeId = aws.ToString(result.Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId)
	volumeId = s.VolumeId
	return
}

var ErrServerNotFound = errors.New("server not found")

/*
Storing a usefull func for generating lists

	node_names:
{{- range $i, $n := .NodeNames }}
  - {{.}}{{end}}
*/

var serverTemplate = template.Must(template.New("vars").Parse(`#!/bin/sh
yum -y install python3 python3-pip python3-wheel
su -c "yes | sudo pip3 install ansible --quiet --exists-action i > /dev/null" {{.User}}
su -c "ansible-galaxy collection install community.docker > /dev/null" {{.User}}

cat <<'EOF' > provision.yml
- name: Initial Provision
  hosts: localhost
  connection: local
  vars:
    buri_base_version: {{.BuriVers}}
    cluster_info: {
{{- range $k, $v := .CInfo}}
      "{{$k}}": "{{$v}}"
{{- end -}}
{{- if .CPorts}}
    }{{- else -}} } {{- end}}
    cluster_name: {{.CName}}
    cluster_ports: {
{{- range $k, $v := .CPorts}}
      "{{$k}}": "{{$v}}"
{{- end -}}
{{- if .CPorts}}
    }{{- else -}} } {{- end}}
    env: {{.Env}}
    env_content: |
      webserver.port=3030
      nerthus.url={{.NUrl}}
      hostname={{.Hostname}}
    hostname: {{.Hostname}}
    nerthus_host: {{.NUrl}}
    os: {{.OS}}
    os_arch: {{.Arch}}
    server_number: {{.ServNum}}
    service: {{.User}}
    system: {{.System}}
    visuale_host: {{.VUrl}}`))

var serverEnd = `  tasks:
  - ansible.builtin.user:
      comment: User for {{ service }}
      home: /home/{{ service }}
      name: '{{ service }}'
    become: "yes"
    become_user: root
    name: Add service user {{ service }}
    register: service_user
  - copy:
      content: |
        #!/bin/sh
        sudo su - {{ service }}
      dest: ~/su_{{ service }}.sh
      mode: u+rwx
    name: Set su file
    when: service != "ec2-user"
  - set_fact:
      arch: amd64
    when: ansible_architecture == "x86_64"
  - set_fact:
      arch: arm64
    when: ansible_architecture == "aarch64"
  - set_fact:
      os: darwin
    when: ansible_facts['os_family'] == "Darwin"
  - become: "yes"
    become_user: root
    name: Check if buri exists
    register: buri
    stat:
      path: /usr/local/bin/buri
  - become: "yes"
    become_user: root
    block:
    - ansible.builtin.get_url:
        dest: /usr/local/bin/
        mode: 493
        url: https://mvnrepo.cantara.no/content/repositories/releases/no/cantara/gotools/buri/v{{
          buri_base_version }}/buri-v{{ buri_base_version }}-{{ os }}-{{ arch }}
      name: Download buri
      register: buri_new
    - file:
        dest: /usr/local/bin/buri
        src: '{{ buri_new.dest }}'
        state: link
      name: Create symbolic link for buri
    when: buri.stat.exists == false
  - become: "yes"
    become_user: ec2-user
    copy:
      content: '{{ env_content }}'
      dest: ~/.env
    name: Set env file
  - become: "yes"
    become_user: root
    block:
    - ansible.builtin.yum:
        name: cronie
        state: latest
      name: Install Cron
    - ansible.builtin.systemd:
        enabled: true
        name: crond
        state: started
      name: Enable Cron
  - become: "yes"
    become_user: '{{ service }}'
    block:
    - copy:
        content: |
          MAILTO=""
          PATH=/bin:/usr/bin:/usr/local/bin
          {{ server_number|int%3*10 }},{{ server_number|int%3*10+30 }} * * * * sudo yum update -y > /dev/null
          0 {{ 3+server_number|int }} * * 6 sudo reboot
          */6 * * * * sudo buri install go -a buri -g no/cantara/gotools > /dev/null
          */6 * * * * sudo buri install go -a nerthus2/probe/health -g no/cantara/gotools > /dev/null
          */6 * * * * buri run go -u -a nerthus2/probe -g no/cantara/gotools > /dev/null
        dest: ~/CRON
        mode: 416
      name: Set service cron file
      register: cron
    - ignore_errors: true
      name: Remove cronjob from crontab scheduler
      shell: crontab -r
      when: cron is changed
    - name: Configure cronjob via crontab scheduler
      shell: crontab ~/CRON
      when: cron is changed
    - name: Start probe
      shell: cd ~ && buri run go -u -a nerthus2/probe -g no/cantara/gotools
EOF
su -c "ansible-playbook provision.yml" `

type ServerData struct {
	BuriVers string
	CInfo    map[string]string
	CName    string
	CPorts   map[string]int
	Env      string
	NUrl     string
	Hostname string
	OS       string
	Arch     string
	ServNum  int
	User     string
	System   string
	VUrl     string
}

func GenServerProv(data ServerData) string {
	w := bytes.Buffer{}
	bW := bufio.NewWriter(base64.NewEncoder(base64.RawStdEncoding, &w))
	err := serverTemplate.Execute(bW, data)
	log.WithError(err).Debug("wrote server template")
	bW.WriteRune('\n')
	bW.WriteString(serverEnd)
	bW.WriteString(data.User)
	log.Info(data.User)
	bW.WriteRune('\n')
	//bW.WriteString("                                      ")
	err = bW.Flush()
	return w.String()
}
