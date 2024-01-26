package actions

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config"
	pconf "github.com/cantara/nerthus2/probe/config"
)

func FileString(cfg pconf.Environment, task config.Task, service string) (err error) {
	dest := replaceAll(cfg, task.Dest)
	if DryRun {
		fmt.Printf("cat > \"%s\" << EOF\n%s\nEOF\n", dest, replaceAll(cfg, task.Text))
		return
	}
	err = writeFile([]byte(replaceAll(cfg, task.Text)), dest)
	if sbragi.WithError(err).Trace("writing file") {
		return
	}
	if !task.Root {
		usr, err := getUser(service)
		if sbragi.WithError(err).Trace("file get user") {
			return err
		}
		err = os.Chown(dest, usr.UID, usr.GID)
		if sbragi.WithError(err).Trace("file change owner") {
			return err
		}
	}
	return
}

func FileBytes(cfg pconf.Environment, task config.Task, service string) (err error) {
	dest := replaceAll(cfg, task.Dest)
	if DryRun {
		fmt.Println("//Will write bytes to file, logically equivilant to next commands example, except not base64 UrlEncoded")
		fmt.Printf("//cat > \"%s\" << EOF\n%s\nEOF\n", dest, base64.URLEncoding.EncodeToString(task.Data))
		return
	}
	err = writeFile(task.Data, dest)
	if sbragi.WithError(err).Trace("writing file") {
		return
	}
	if !task.Root {
		usr, err := getUser(service)
		if sbragi.WithError(err).Trace("file get user") {
			return err
		}
		err = os.Chown(dest, usr.UID, usr.GID)
		if sbragi.WithError(err).Trace("file change owner") {
			return err
		}
	}
	return
}

func writeFile(data []byte, dest string) error {
	err := os.MkdirAll(filepath.Dir(dest), 0750)
	if sbragi.WithError(err).Trace("making dirs to download dest", "dest", dest) {
		return err
	}
	f, err := os.OpenFile(dest, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0640)
	if sbragi.WithError(err).Trace("opening dest file", "dest", dest) {
		return err
	}
	defer sbragi.WithErrorFunc(f.Close).Trace("closing dest file", "dest", dest)
	err = io.ErrShortWrite
	l := len(data)
	tot := 0
	for err == io.ErrShortWrite && tot < l {
		var n int
		n, err = f.Write(data[tot:])
		tot += n
	}
	if sbragi.WithError(err).Trace("writing data to file") {
		return err
	}
	return nil
}
