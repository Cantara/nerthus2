package schema

import (
	"embed"
	_ "embed"
	"fmt"
	"io"
	iofs "io/fs"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"

	"github.com/cantara/bragi/sbragi"
)

//go:embed *.cue */*
var fs embed.FS

type rt struct {
	CurrentDirectory string `json:"currentDirectory"`
}

func Load(dir string, files []string, dest any) error {
	return LoadFS(dir, files, fs, dest)
}
func LoadFS(dir string, files []string, fs iofs.FS, dest any) error {
	sbragi.Info("loading", "dir", dir, "files", files)
	overlay := make(map[string]load.Source)
	err := iofs.WalkDir(fs, ".", func(path string, d iofs.DirEntry, err error) error {
		sbragi.Info("reading fs", "path", path)
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".cue" {
			return nil
		}
		file, err := fs.Open(path)
		if err != nil {
			return err
		}
		bytes, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(path, "_schema") {
			path = filepath.Join("_schema", path)
		}
		path = filepath.Join(dir, path)
		overlay[path] = load.FromBytes(bytes)
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}
	configInst := load.Instances(files, &load.Config{
		Dir:     dir,
		Package: "*",
		Overlay: overlay,
	})[0]

	sbragi.Info("loaded instances")
	if err := configInst.Err; err != nil {
		return fmt.Errorf("cannot load configuration from %q: %v", configInst.Root, err)
	}
	ctx := cuecontext.New()
	configVal := ctx.BuildInstance(configInst)
	fields, err := configVal.Fields()
	sbragi.WithError(err).Trace("built instance and got fields")
	//a, d := configVal.Struct()
	//i := a.Fields()
	for fields.Next() {
		fmt.Println(fields.Label(), fields.Value())
	}
	if err := configVal.Validate(cue.All()); err != nil {
		return fmt.Errorf("invalid configuration from %q: %v", dir, errors.Details(err, nil))
	}

	if err := configVal.Decode(dest); err != nil {
		return fmt.Errorf("cannot decode final configuration: %v", errors.Details(err, nil))
	}
	return nil
}
