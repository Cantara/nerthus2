package actions

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	pconf "github.com/cantara/nerthus2/probe/config"
)

var linkInfo = "Link to file"

func Link(cfg pconf.Environment, task config.Task, service string) (err error) {
	source, dest := replaceAll(cfg, task.Source), replaceAll(cfg, task.Dest)
	/*
		err = Command(cfg, cmd.Task{
			Info:    linkInfo,
			Type:    "command",
			Command: fmt.Sprintf("ln -s \"%s\" \"%s\"", source, dest),
			Root:    task.Root,
		}, service)
		if sbragi.WithError(err).Trace(linkInfo) {
			return
		}
	*/
	fi, err := os.Lstat(source)
	if err == nil {
		if fi.Mode()&fs.ModeSymlink == 0 {
			err = fmt.Errorf("file exists and is not symlink, paht=%s", source)
			return
		}
		c := exec.Command("readlink", source)
		out, err := c.CombinedOutput()
		if sbragi.WithError(err).Trace("read link", "path", source, "cmd", c.String()) {
			return err
		}
		if strings.TrimSpace(string(out)) == dest {
			sbragi.Trace("link allready points to dest, no action needed", "source", source, "dest", dest)
			return nil
		}
		sbragi.WithError(os.Remove(source)).Trace("removing old link", "source", source)
	}
	err = os.Link(source, dest)
	if sbragi.WithError(err).Trace(linkInfo) {
		return
	}
	if !task.Root {
		usr, err := getUser(service)
		if sbragi.WithError(err).Trace(linkInfo + ": get user") {
			return err
		}
		err = os.Chown(dest, usr.UID, usr.GID)
		if sbragi.WithError(err).Trace(linkInfo) {
			return err
		}
	}

	return
}
