package actions

import (
	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	pconf "github.com/cantara/nerthus2/probe/config"
)

var enableInfo = "Enable service"

func Enable(cfg pconf.Environment, task config.Task, service string) (err error) {
	command := []string{"systemctl", "enable", task.Service}
	if task.Start {
		command = append(command, "--now")
	}
	err = Command(cfg, config.Task{
		Info:    enableInfo,
		Type:    "command",
		Command: command,
		Root:    task.Root,
	}, service)
	if sbragi.WithError(err).Trace(enableInfo) {
		return
	}
	return
}
