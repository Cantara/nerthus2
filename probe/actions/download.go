package actions

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	pconf "github.com/cantara/nerthus2/probe/config"
)

func Download(cfg pconf.Environment, task config.Task, service string) (err error) {
	source := replaceAll(cfg, task.Source)
	dest := replaceAll(cfg, task.Dest)
	if DryRun {
		fmt.Printf("curl \"%s\" > \"%s\"\n", source, dest)
		return
	}
	err = download(source, dest)
	sbragi.WithError(err).Trace("downloading directly to dest")
	if !task.Root {
		usr, err := getUser(service)
		if sbragi.WithError(err).Trace("download get user") {
			return err
		}
		err = os.Chown(dest, usr.UID, usr.GID)
		if sbragi.WithError(err).Trace("download change owner") {
			return err
		}
	}
	/*
		if task.Root {
			tmp := fmt.Sprintf("%s%s", os.TempDir(), filepath.Base(dest))
			if DryRun {
				fmt.Printf("sudo curl \"%s\" > \"%s\"\n", source, tmp)
			} else {
				err = download(source, tmp)
				if sbragi.WithError(err).Trace("downloading to tmp file") {
					return
				}
			}
			info := "change owner of tmp file"
			err = Command(cfg, cmd.Task{
				Info:    info,
				Type:    "command",
				Command: fmt.Sprintf("sudo chown root \"%s\"", tmp),
				Root:    task.Root,
			}, service)
			if sbragi.WithError(err).Trace(info) {
				return
			}
			info = "change group of tmp file"
			err = Command(cfg, cmd.Task{
				Info:    info,
				Type:    "command",
				Command: fmt.Sprintf("sudo chgrp root \"%s\"", tmp),
				Root:    task.Root,
			}, service)
			if sbragi.WithError(err).Trace(info) {
				return
			}
			info = "move tmp file to final location"
			err = Command(cfg, cmd.Task{
				Info:    info,
				Type:    "command",
				Command: fmt.Sprintf("sudo mv \"%s\" \"%s\"", tmp, dest),
				Root:    task.Root,
			}, service)
			if sbragi.WithError(err).Trace(info) {
				return
			}
		}
	*/

	return
}

func download(source, dest string) error {
	err := os.MkdirAll(filepath.Dir(dest), 0750)
	if sbragi.WithError(err).Trace("making dirs to download dest", "dest", dest) {
		return err
	}
	f, err := os.OpenFile(dest, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0640)
	if sbragi.WithError(err).Trace("opening dest file", "dest", dest) {
		return err
	}
	defer sbragi.WithErrorFunc(f.Close).Trace("closing dest file", "dest", dest)

	resp, err := http.Get(source)
	if sbragi.WithError(err).Trace("getting file", "source", source, "dest", dest) {
		return err
	}
	defer resp.Body.Close()

	n, err := io.Copy(f, resp.Body)
	if sbragi.WithError(err).Trace("storing file", "source", source, "dest", dest, "size", n) {
		return err
	}

	return nil
}
