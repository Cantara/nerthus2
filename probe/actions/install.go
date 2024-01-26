package actions

import (
	"fmt"
	"strings"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	pconf "github.com/cantara/nerthus2/probe/config"
)

var installInfo = "install package"

func Install(cfg pconf.Environment, task config.Task, service string) (err error) {
	for _, manager := range cfg.System.Cluster.Node.Os.PackageManagers {
		if config.Contains(task.Package.Managers, manager.Name) < 0 {
			continue
		}
		c := make([]string, len(manager.Syntax))
		for i := range manager.Syntax {
			c[i] = strings.ReplaceAll(manager.Syntax[i], "<package>", task.Package.Name)
		}
		err = Command(cfg, config.Task{
			Info:    installInfo,
			Type:    "command",
			Command: c,
			Root:    task.Root || manager.Root,
		}, service)
		if sbragi.WithError(err).Trace(installInfo) {
			return
		}
		return
	}
	err = fmt.Errorf("required manager not available, %s=%v, %s=%s", "managers", cfg.System.Cluster.Node.Os.PackageManagers, "required", task.Manager)
	return
}
