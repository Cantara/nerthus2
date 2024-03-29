package aws

/*
import (
	"embed"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	log "github.com/cantara/bragi"
	databaselib "github.com/cantara/nerthus2/cloud/aws/database"
	"github.com/cantara/nerthus2/cloud/aws/key"
	keylib "github.com/cantara/nerthus2/cloud/aws/key"
	loadbalancerlib "github.com/cantara/nerthus2/cloud/aws/loadbalancer"
	"github.com/cantara/nerthus2/cloud/aws/security"
	securitylib "github.com/cantara/nerthus2/cloud/aws/security"
	serverlib "github.com/cantara/nerthus2/cloud/aws/server"
	"github.com/cantara/nerthus2/cloud/aws/tag"
	vpclib "github.com/cantara/nerthus2/cloud/aws/vpc"
	"github.com/cantara/nerthus2/slack"
)

type Health struct {
	Name string `form:"service_name" json:"service_name" xml:"service_name"`
	Tag  string `form:"service_tag" json:"service_tag" xml:"service_tag"`
	Type string `form:"service_type" json:"service_type" xml:"service_type"`
}

type Service struct {
	Port             int    `form:"port" json:"port" xml:"port" binding:"required"`
	Path             string `form:"path" json:"path" xml:"path" binding:"required"`
	Icon             string `form:"icon" json:"icon" xml:"icon"`
	ELBListenerArn   string `form:"elb_listener_arn" json:"elb_listener_arn" xml:"elb_listener_arn" binding:"required"`
	ELBSecurityGroup string `form:"elb_securitygroup_id" json:"elb_securitygroup_id" xml:"elb_securitygroup_id"`
	UpdateProp       string `form:"semantic_update_service_properties" json:"semantic_update_service_properties" xml:"semantic_update_service_properties"`
	ArtifactId       string `form:"artifact_id" json:"artifact_id" xml:"artifact_id" binding:"required"`
	LocalOverride    string `form:"local_override_properties" json:"local_override_properties" xml:"local_override_properties"`
	Health           Health `form:"health" json:"health" xml:"health"`
	Key              string `form:"key" json:"key" xml:"key"`
}

func (c AWS) AddServiceToServer(scope, serverName string, v vpclib.VPC, k key.Key, sg security.Group, slackId string, service Service) (message string) {
	seq := sequence{
		ec2:           c.ec2,
		elb:           c.elb,
		shouldCleanUp: false,
		deleters:      NewStack(),
		slackId:       slackId,
		scope:         scope,
		service:       service,
		vpc:           v,
		key:           k,
		securityGroup: sg,
	}
	defer seq.Cleanup()
	//Get server from server name
	s, err := serverlib.GetServer(serverName, scope, k, sg, c.ec2)
	if err != nil {
		log.AddError(err).Fatal("While getting server by name")
	}
	seq.server = s

	//AWS
	isNotNewService, err := CheckIfServiceExcistsInScope(scope, service.ArtifactId, c.ec2)
	if err != nil {
		log.AddError(err).Fatal("While chekking if service exits in scope")
	}
	if isNotNewService {
		seq.GetTargetGroup()
	} else {
		seq.AddLoadbalancerAuthorizationToSecurityGroup()
		seq.CreateTargetGroup()
	}
	seq.CreateTarget()
	if !isNotNewService {
		seq.AddRuleToListener()
	}

	if isNotNewService {
		seq.TagAdditionalServer()
	} else {
		seq.TagNewService()
	}
	seq.InstallOnServer()

	seq.SendServiceOnServer()
	seq.FinishedAllOpperations()
	//cryptData = seq.cryptData
	message = "succsess"
	return
}

func (c AWS) AddServerToScope(scope, serverName string, v vpclib.VPC, k key.Key, sg security.Group, slackId string) (message string) {
	seq := sequence{
		ec2:           c.ec2,
		elb:           c.elb,
		shouldCleanUp: false,
		deleters:      NewStack(),
		slackId:       slackId,
		scope:         scope,
		vpc:           v,
		key:           k,
		securityGroup: sg,
	}
	defer seq.Cleanup()

	//AWS
	seq.CheckServerName(serverName)
	seq.StartingServiceSettup()
	seq.CreateNewServer(serverName)
	seq.WaitForServerToStart()
	seq.VerifyServerSSH()
	seq.AddAutoUpdate()
	seq.InstallFilebeat()
	/*
		isNotNewService, err := CheckIfServiceExcistsInScope(scope, service.ArtifactId, c.ec2)
		if err != nil {
			log.AddError(err).Fatal("While chekking if service exits in scope")
		}
		if isNotNewService {
			seq.GetTargetGroup()
		} else {
			seq.CreateTargetGroup()
		}
		seq.CreateTarget()
		if !isNotNewService {
			seq.AddRuleToListener()
		}
		seq.DoneSettingUpServer()

		if isNotNewService {
			seq.TagAdditionalServer()
		} else {
			seq.TagNewService()
		}
		seq.InstallOnServer()
*/ /*
	seq.SendLogin()
	seq.FinishedAllOpperations()
	message = "succsess"
	return
}

func (c AWS) CreateDatabase(scope, artifactId string, v vpclib.VPC, sg security.Group, slackId string) (endpoint string) {
	seq := sequence{
		ec2:           c.ec2,
		rds:           c.rds,
		shouldCleanUp: false,
		deleters:      NewStack(),
		slackId:       slackId,
		scope:         scope,
		vpc:           v,
		securityGroup: sg,
	}
	defer seq.Cleanup()

	//AWS
	seq.CreateDBSecurityGroup(artifactId)
	seq.CreateNewDatabase(artifactId)

	seq.SendDBSettup()
	seq.FinishedAllOpperations()
	endpoint = seq.database.Endpoint
	return
}

func (c AWS) CreateScope(scope string) (cryptData string) {
	seq := sequence{
		ec2:           c.ec2,
		elb:           c.elb,
		shouldCleanUp: false,
		deleters:      NewStack(),
		scope:         scope,
	}
	defer seq.Cleanup()

	//AWS
	seq.StartingServerSettup()
	seq.CreateKey()
	seq.GetVPC()
	seq.CreateSecurityGroup()

	seq.SendScope()
	seq.FinishedAllOpperations()
	cryptData = seq.cryptData
	return
}

type sequence struct {
	ec2             *ec2.Client
	elb             *elbv2.Client
	rds             *rds.Client
	shouldCleanUp   bool
	deleters        Stack
	slackId         string
	scope           string
	service         Service
	key             keylib.Key
	PemName         string
	cryptData       string
	vpc             vpclib.VPC
	securityGroup   securitylib.Group
	dbSecurityGroup securitylib.Group
	database        databaselib.Database
	server          serverlib.Server
	targetGroup     loadbalancerlib.TargetGroup
	rule            loadbalancerlib.Rule
	serversh        servershlib.Server
	user            servershlib.User
}

func (c *sequence) Cleanup() {
	if a := recover(); a != nil {
		log.Warning("Recovered: ", a)
		c.shouldCleanUp = true
	}
	if !c.shouldCleanUp {
		return
	}
	log.Info("Cleanup started.")
	c.cryptData = ""
	slack.SendStatus(":x: Something went wrong starting cleanup.")
	for delFunc := c.deleters.Pop(); delFunc != nil; delFunc = c.deleters.Pop() {
		delFunc()
	}
	log.Info("Cleanup is \"done\", exiting.")
	slack.SendStatus(":heavy_check_mark: Cleanup is \"done\".")
}

func (c sequence) CheckServerName(name string) {
	available, err := serverlib.NameAvailable(name, c.ec2)
	if err != nil {
		log.AddError(err).Fatal("While checking server name availablility")
	}
	if !available {
		log.Fatal("Servername is not available")
	}
}

func (c sequence) StartingServerSettup() {
	s := fmt.Sprintf("%s: %s Starting to setup server in aws.", c.scope, c.service.ArtifactId)
	log.Info(s)
	slack.SendStatus(s)
}

func (c sequence) StartingServiceSettup() {
	s := fmt.Sprintf("%s: %s Starting to setup service on server in aws.", c.scope, c.service.ArtifactId)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) CreateKey() {
	// Create a new key
	key, err := keylib.NewKey(c.scope, c.ec2)
	_, err = key.Create()
	if err != nil {
		log.AddError(err).Fatal("While creating keypair")
	}
	c.deleters.Push(cleanup("Key pair", "while deleting created key pair", &key))
	s := fmt.Sprintf("%s: Created key pair %s %s", c.scope, key.Name, key.Fingerprint)
	log.Info(s)
	slack.SendStatus(s)
	pem, err := os.OpenFile("./"+key.PemName, os.O_WRONLY|os.O_CREATE, 0600)
	if err == nil {
		fmt.Fprint(pem, key.Material)
		pem.Close()
	}
	c.key = key
	c.PemName = c.key.PemName
}

func (c *sequence) GetVPC() {
	// Get a list of VPCs so we can associate the group with the first VPC.
	vpc, err := vpclib.GetVPC(c.ec2)
	if err != nil {
		log.AddError(err).Fatal("While getting vpcId")
	}
	s := fmt.Sprintf("%s: Found VPCId: %s.", c.scope, vpc.Id)
	log.Info(s)
	slack.SendStatus(s)
	c.vpc = vpc
}

func (c *sequence) CreateSecurityGroup() {
	securityGroup, err := securitylib.NewGroup(c.scope, c.vpc, c.ec2)
	_, err = securityGroup.Create()
	if err != nil {
		log.AddError(err).Fatal("While creating security group")
	}
	c.deleters.Push(cleanup("Security group", "while deleting created security group",
		&securityGroup))
	s := fmt.Sprintf("%s: Created security group %s with VPC %s.",
		c.scope, securityGroup.Id, c.vpc.Id)
	log.Info(s)
	slack.SendStatus(s)
	c.securityGroup = securityGroup
	c.AddBaseAuthorizationToSecurityGroup()
}

func (c *sequence) CreateDBSecurityGroup(artifactId string) {
	securityGroup, err := securitylib.NewDBGroup(servershlib.ToFriendlyName(artifactId), c.scope, c.vpc, c.ec2)
	_, err = securityGroup.Create()
	if err != nil {
		log.AddError(err).Fatal("while creating security group")
	}
	c.deleters.Push(cleanup("Security group", "while deleting created security group",
		&securityGroup))
	s := fmt.Sprintf("%s: Created security group %s with VPC %s.",
		c.scope, securityGroup.Id, c.vpc.Id)
	log.Info(s)
	slack.SendStatus(s)
	c.dbSecurityGroup = securityGroup
	c.AddDatabaseAuthorizationToSecurityGroup()
}

func (c *sequence) AddBaseAuthorizationToSecurityGroup() {
	err := c.securityGroup.AddBaseAuthorization()
	if err != nil {
		log.AddError(err).Fatal("Could not add base authorization")
	}
	s := fmt.Sprintf("%s: Added base authorization to security group: %s.", c.scope, c.securityGroup.Id)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) AddDatabaseAuthorizationToSecurityGroup() {
	err := c.dbSecurityGroup.AddDatabaseAuthorization(c.securityGroup.Id)
	if err != nil {
		log.AddError(err).Fatal("Could not add database authorization")
	}
	s := fmt.Sprintf("%s: Added database authorization to security group: %s.", c.scope, c.dbSecurityGroup.Id)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) AddLoadbalancerAuthorizationToSecurityGroup() {
	err := c.securityGroup.AddLoadbalancerAuthorization(c.service.ELBSecurityGroup, c.service.Port)
	if err != nil {
		log.AddError(err).Fatal("Could not add base authorization")
	}
	s := fmt.Sprintf("%s: %s %s, Added base authorization to security group: %s.", c.scope, c.server.Name, c.service.ArtifactId, c.securityGroup.Id)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) CreateNewDatabase(artifactId string) {
	database, err := databaselib.NewDatabase(servershlib.ToFriendlyName(artifactId), c.scope, c.securityGroup, c.rds)
	_, err = database.Create()
	if err != nil {
		log.Fatal("Could not create database", err)
	}
	c.deleters.Push(cleanup("Database", "while deleting created database", &database))
	s := fmt.Sprintf("%s: Created database: %s.", c.scope, database.ARN)
	log.Info(s)
	slack.SendStatus(s)
	c.database = database
}

func (c *sequence) CreateNewServer(serverName string) {
	server, err := serverlib.NewServer(serverName, c.scope, c.key, c.securityGroup, c.ec2)
	_, err = server.Create()
	if err != nil {
		log.Fatal("Could not create server", err)
	}
	c.deleters.Push(cleanup("Server", "while deleting created server", &server))
	s := fmt.Sprintf("%s: %s, Created server: %s.", c.scope, c.server.Name, server.Id)
	log.Info(s)
	slack.SendStatus(s)
	c.server = server
}

func (c *sequence) WaitForServerToStart() {
	err := c.server.WaitUntilRunning()
	s := fmt.Sprintf("%s: %s, Server %s is now in running state.", c.scope, c.server.Name, c.server.Id)
	log.Info(s)
	slack.SendStatus(s)
	_, err = c.server.GetPublicDNS()
	if err != nil {
		log.AddError(err).Fatal("While getting public dns name")
	}
	s = fmt.Sprintf("%s: %s, Got server %s's public dns %s.", c.scope, c.server.Name, c.server.Id, c.server.PublicDNS)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) CreateTargetGroup() {
	targetGroup, err := loadbalancerlib.NewTargetGroup(c.scope, c.service.ArtifactId, c.service.Path, c.service.Port, c.vpc, c.elb)
	_, err = targetGroup.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While creating target group for %s", c.server.Name))
	}
	c.deleters.Push(cleanup("Target group", "while deleting created target group", &targetGroup))
	s := fmt.Sprintf("%s: %s %s, Created target group: %s.", c.scope, c.server.Name, c.service.ArtifactId, targetGroup.ARN)
	log.Info(s)
	slack.SendStatus(s)
	c.targetGroup = targetGroup
}

func (c *sequence) GetTargetGroup() {
	targetGroup, err := loadbalancerlib.GetTargetGroup(c.scope, c.service.ArtifactId, c.service.Path, c.service.Port, c.elb)
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While getting target group for %s", c.server.Name))
	}
	s := fmt.Sprintf("%s: %s %s, Got target group: %s.", c.scope, c.server.Name, c.service.ArtifactId, targetGroup.ARN)
	log.Info(s)
	slack.SendStatus(s)
	c.targetGroup = targetGroup
}

func (c *sequence) CreateTarget() {
	target, err := loadbalancerlib.NewTarget(c.targetGroup, c.server, c.elb)
	_, err = target.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While adding target to target group %s", c.targetGroup.ARN))
	}
	c.deleters.Push(cleanup("Target in targetgroup", "while removing registered target from targetgroup", &target))
	s := fmt.Sprintf("%s: %s %s, Registered server %s as target for target group %s.", c.scope, c.server.Name, c.service.ArtifactId, c.server.Id, c.targetGroup.ARN)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) AddRuleToListener() {
	listener, err := loadbalancerlib.GetListener(c.service.ELBListenerArn, c.elb)
	rule, err := loadbalancerlib.NewRule(listener, c.targetGroup, c.elb)
	_, err = rule.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While adding rule to elb %s", listener.ARN))
	}
	c.deleters.Push(cleanup("Rule", "while removing rule added to loadbalancer", &rule))
	s := fmt.Sprintf("%s: %s %s, Adding elastic load balancer rule: %s.", c.scope, c.server.Name, c.service.ArtifactId, rule.ARN)
	log.Info(s)
	slack.SendStatus(s)
	c.rule = rule
}

func (c *sequence) TagNewService() {
	VolumeId, err := c.server.GetVolumeId()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While getting volume id for server %s", c.server.Name))
	}
	listener, err := loadbalancerlib.GetListener(c.service.ELBListenerArn, c.elb)
	loadbalancerARN, err := listener.GetLoadbalancer()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While getting loadbalancerARN for listener %s", c.service.ELBListenerArn))
	}
	t, err := tag.NewNewTag(c.service.ArtifactId, c.scope, c.key.Id, c.securityGroup.Id, c.server.Id, VolumeId, c.server.NetworkInterfaceId, c.server.ImageId,
		c.targetGroup.ARN, c.rule.ARN, c.service.ELBListenerArn, loadbalancerARN, c.ec2, c.elb)
	_, err = t.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While tagging new service %s", c.service.ArtifactId))
	}
	c.deleters.Push(cleanup("Tag", "while removing tag added to all resources used by service", &t))
	s := fmt.Sprintf("%s: %s %s, Adding tag to all resources used by service: %s.", c.scope, c.server.Name, c.service.ArtifactId, c.service.ArtifactId)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) TagAdditionalServer() {
	VolumeId, err := c.server.GetVolumeId()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While getting volume id for server %s", c.server.Name))
	}
	t, err := tag.NewAddTag(c.service.ArtifactId, c.scope, c.server.Id, VolumeId, c.server.NetworkInterfaceId, c.server.ImageId, c.ec2)
	_, err = t.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While tagging additional service %s", c.service.ArtifactId))
	}
	c.deleters.Push(cleanup("Tag", "while removing tag added to resources used by the additional service", &t))
	s := fmt.Sprintf("%s: %s %s, Adding tag to resources used by additional service: %s.", c.scope, c.server.Name, c.service.ArtifactId, c.service.ArtifactId)
	log.Info(s)
	slack.SendStatus(s)
}

func (c sequence) DoneSettingUpServer() {
	s := fmt.Sprintf("%s: %s %s, Done setting up server in aws %s.", c.scope, c.server.Name, c.service.ArtifactId, c.server.Id)
	log.Info(s)
	slack.SendStatus(s)
}

func (c sequence) WaitForELBRuleToBeHealthy() {
	s := fmt.Sprintf("%s: %s %s, Started waiting for elb rule to be healthy %s.", c.scope, c.server.Name, c.service.ArtifactId, c.rule.ARN)
	log.Info(s)
	slack.SendStatus(s)
	time.Sleep(30 * time.Second)
	s = fmt.Sprintf("%s: %s %s, Done waiting for elb rule to be healthy %s.", c.scope, c.server.Name, c.service.ArtifactId, c.rule.ARN)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) VerifyServerSSH() {
	serv, _ := servershlib.NewServer(c.server.PublicDNS, c.key.PemName)
	c.serversh = serv
	err := c.serversh.WaitForConnection()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While waiting for connection for %s: %s", c.server.Name, c.server.PublicDNS))
	}
	s := fmt.Sprintf("%s: %s, SSH connection to %s is verified.", c.scope, c.server.Name, c.server.PublicDNS)
	log.Info(s)
	slack.SendStatus(s)
}

func (c sequence) StartingServiceInstallation() {
	time.Sleep(time.Second * 30)
	s := fmt.Sprintf("%s: %s %s, Starting to install stuff on server %s.", c.scope, c.server.Name, c.service.ArtifactId, c.server.PublicDNS)
	log.Info(s)
	slack.SendStatus(s)
}

func (c sequence) AddAutoUpdate() {
	err := c.serversh.AddAutoUpdate()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While adding auto updatating %s: %s", c.server.Name, c.server.PublicDNS))
	}
	s := fmt.Sprintf("%s: %s, Adding auto update to server %s.", c.scope, c.server.Name, c.server.PublicDNS)
	log.Info(s)
	slack.SendStatus(s)
}

func (c sequence) UpdateServer() {
	err := c.serversh.Update()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While updatating %s: %s", c.server.Name, c.server.PublicDNS))
	}
	s := fmt.Sprintf("%s: %s %s, Updated server %s.", c.scope, c.server.Name, c.service.ArtifactId, c.server.PublicDNS)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) InstallFilebeat() {
	filebeat, err := servershlib.NewFilebeat(os.Getenv("filebeat_password"), c.serversh)
	_, err = filebeat.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While installed filebeat on %s", c.server.Name))
	}
	c.deleters.Push(cleanup("Filebeat service from server", "while removing filebeat service from server", &filebeat))
	s := fmt.Sprintf("%s: %s, Added filebeat.", c.scope, c.server.Name)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) InstallPrograms() {
	java, err := servershlib.NewJava(servershlib.JAVA_ONE_ELEVEN, c.serversh)
	_, err = java.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While verifying or installing java %s", servershlib.JAVA_ONE_ELEVEN))
	}
	c.deleters.Push(cleanup("Java from server", "while removing java if it was installed", &java))
	s := fmt.Sprintf("%s: %s %s, Verified or installed java %s.", c.scope, c.server.Name, c.service.ArtifactId, servershlib.JAVA_ONE_ELEVEN)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) AddUser() {
	user, err := servershlib.NewUser(c.service.ArtifactId, c.serversh)
	_, err = user.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While adding user %s", user.Name))
	}
	c.deleters.Push(cleanup("User from server", "while removing user from server", &user))
	s := fmt.Sprintf("%s: %s %s, Added user %s.", c.scope, c.server.Name, c.service.ArtifactId, user.Name)
	log.Info(s)
	slack.SendStatus(s)
	c.user = user
}

func (c *sequence) AddFilebeatService() {
	filebeatService, err := servershlib.NewFilebeatService(c.server.Name, c.service.ArtifactId, c.user.Name, c.serversh)
	_, err = filebeatService.Create()
	if err != nil {
		log.AddError(err).Fatal(fmt.Sprintf("While adding filebeat script for %s", c.service.ArtifactId))
	}
	c.deleters.Push(cleanup("Filebeat service from server", "while removing filebeat service from server", &filebeatService))
	s := fmt.Sprintf("%s: %s %s, Added filebeat service.", c.scope, c.server.Name, c.service.ArtifactId)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) InstallService() {
	healthReportUrl := fmt.Sprintf("%s/%s/%s?service_tag=%s&service_type=%s", os.Getenv("health_url_with_base_path"), url.PathEscape(c.service.Health.Name), c.server.Name, url.QueryEscape(c.service.Health.Tag), url.QueryEscape(c.service.Health.Type))
	service, err := servershlib.NewService(c.service.ArtifactId, c.service.UpdateProp, c.service.LocalOverride, healthReportUrl, c.service.Path, c.service.Icon, c.service.Port, c.user, c.serversh)
	_, err = service.Create()
	if err != nil {
		log.AddError(err).Fatal("While setting up service in user")
	}
	c.deleters.Push(cleanup("Service installed on server", "while stopping service", &service))
	s := fmt.Sprintf("%s: %s %s, Done installing service on server %s.", c.scope, c.server.Name, c.service.ArtifactId, c.server.PublicDNS)
	log.Info(s)
	slack.SendStatus(s)
}

func (c *sequence) SendScope() {
	slackId, err := slack.SendBase(fmt.Sprintf("Created new scope: %s", c.scope))
	if err != nil {
		log.AddError(err).Fatal("While sending encrypted cert and login to slack")
	}
	encrypted, err := Encrypt(c.scope, c.vpc, c.key, c.securityGroup, slackId)
	if err != nil {
		log.AddError(err).Fatal("While encrypting data to send to slack")
	}
	c.cryptData = encrypted
	_, err = slack.SendFollowup(fmt.Sprintf("%s\n```%s```", c.key.PemName, encrypted), slackId)
	if err != nil {
		log.AddError(err).Fatal("While sending encrypted cert to slack")
	}
}

//go:embed ssh_base.sh
var fsSSH embed.FS

func (c *sequence) SendLogin() {
	script, err := fsSSH.ReadFile("ssh_base.sh")
	if err != nil {
		log.AddError(err).Fatal("While reading in base ssh script")
	}
	scripts := strings.ReplaceAll(string(script), "<url>", os.Getenv("url"))
	scripts = strings.ReplaceAll(scripts, "<key>", c.key.Material)
	scripts = strings.ReplaceAll(scripts, "<server>", c.server.Name)
	_, err = slack.SendFollowupWFile(fmt.Sprintf("%s.sh", c.server.Name), fmt.Sprintf("%s\n`ssh ec2-user@%s -i %s`", c.server.Name, c.server.PublicDNS, c.key.PemName), c.slackId, []byte(scripts))
	if err != nil {
		log.AddError(err).Fatal("While sending new server login to slack")
	}
}

func (c *sequence) SendServiceOnServer() {
	_, err := slack.SendFollowup(fmt.Sprintf("%s > %s", c.service.ArtifactId, c.server.Name), c.slackId)
	if err != nil {
		log.AddError(err).Fatal("While sending info about new service on server to slack")
	}
}

func (c *sequence) SendDBSettup() {
	_, err := slack.SendFollowup(fmt.Sprintf("> Database %s\n ```Endpoint: %s\nDatabase: %s\nUsername: %[3]s\nPassword: %s```", c.database.Name, c.database.Endpoint, c.database.Database, c.database.Password), c.slackId)
	if err != nil {
		log.AddError(err).Fatal("While sending database settup to slack")
	}
}

/*
func (c *sequence) SendCertLogin() {
	encrypted, err := Encrypt(c.scope, c.vpc, c.key, c.securityGroup)
	if err != nil {
		log.AddError(err).Fatal("While encrypting data to send to slack")
	}
	c.cryptData = encrypted
	err = slack.SendServer(fmt.Sprintf(" `ssh ec2-user@%s -i %s`\n%s\n```%s```", c.server.PublicDNS, c.key.PemName, c.service.ArtifactId, encrypted))
	if err != nil {
		log.AddError(err).Fatal("While sending encrypted cert and login to slack")
	}
}
*/ /*

func (c *sequence) FinishedAllOpperations() {
	s := fmt.Sprintf("%s: %s %s, Completed all operations for creating the new server %s.", c.scope, c.server.Name, c.service.ArtifactId, c.server.Name)
	log.Info(s)
	slack.SendStatus(s)
	//shouldCleanUp = true
	return
}

func (seq *sequence) InstallOnServer() {
	//Server
	seq.WaitForELBRuleToBeHealthy()
	seq.StartingServiceInstallation()
	seq.VerifyServerSSH()
	seq.UpdateServer()
	seq.InstallPrograms()
	seq.AddUser()
	seq.InstallService()
	seq.AddFilebeatService()
}

/*
Hazelcast stuff, will readd you later

	if false { // Enable hazelcast
		err = securityGroup.AuthorizeHazelcast()
		if err != nil {
			log.AddError(err).Fatal("Could not add hazelcast authorization")
		}
		s = fmt.Sprintf("Added hazelcast authorization to security group: %s.", securityGroup.Id)
		log.Info(s)
		slack.SendStatus(s)
	}
*/
