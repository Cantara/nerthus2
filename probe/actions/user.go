package actions

import (
	"os/user"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	pconf "github.com/cantara/nerthus2/probe/config"
)

var userInfo = "user to file"

func Useradd(cfg pconf.Environment, task config.Task, service string) (err error) {
	_, err = user.Lookup(task.Username)
	if !sbragi.WithError(err).Trace("getting user") {
		return nil
	}
	err = Command(cfg, config.Task{
		Type:    "command",
		Info:    task.Info,
		Root:    task.Root,
		Command: []string{"useradd", task.Username, "-m"},
	}, service)
	return
}
