package actions

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	pconf "github.com/cantara/nerthus2/probe/config"
)

func Command(cfg pconf.Environment, task config.Task, service string) error {
	if DryRun {
		if task.Root {
			fmt.Print("sudo ")
		}
		fmt.Println(replaceAllAll(cfg, task.Command))
		return nil
	}
	cmd := replaceAllAll(cfg, task.Command)
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Env = []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
	}
	if IsRoot && !task.Root {
		c.SysProcAttr = getSysProcAttr(service)
		if c.SysProcAttr == nil {
			return fmt.Errorf("got nil SysProcAttr when trying to set perms for user execution, system=%s service=%s", os.Getenv("SUDO_USER"), service)
		}
		u, _ := getUser(service)
		c.Env = append(c.Env, fmt.Sprintf("HOME=%s", u.Home))
	} else {
		c.Env = append(c.Env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
	}
	out, err := c.CombinedOutput()
	if sbragi.WithError(err).Info("executed command", "out", out, "command", c.String()) {
		return err
	}
	return nil
}

type User struct {
	Username string
	UID      int
	GID      int
	Home     string
}

var Users = make(map[string]User)

var SysProcAttr = make(map[string]*syscall.SysProcAttr)

func getUser(service string) (usr User, err error) {
	if service == "" {
		service = os.Getenv("SUDO_USER")
	}
	var ok bool
	usr, ok = Users[service]
	if !ok {
		u, err := user.Lookup(service)
		if sbragi.WithError(err).Trace("getting service user info") {
			return usr, err
		}
		uid, err := strconv.Atoi(u.Uid)
		if sbragi.WithError(err).Trace("getting service user uid") {
			return usr, err
		}
		gid, err := strconv.Atoi(u.Gid)
		if sbragi.WithError(err).Trace("getting service user gid") {
			return usr, err
		}
		usr = User{
			Username: u.Username,
			UID:      uid,
			GID:      gid,
			Home:     u.HomeDir,
		}
		Users[service] = usr

	}
	return usr, nil
}
func getSysProcAttr(service string) *syscall.SysProcAttr {
	usr, err := getUser(service)
	if sbragi.WithError(err).Trace("getting user info") {
		return nil
	}
	return &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(usr.UID),
			Gid: uint32(usr.GID),
		},
		//Chroot: u.HomeDir,
		Setsid: true,
		//Setpgid: true,
		//Noctty: true,
	}
}
