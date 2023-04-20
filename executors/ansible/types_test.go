package ansible

import (
	"os"
	"testing"
)

func TestReadRoleDir(t *testing.T) {
	roles := make(map[string]Role)
	nerthusFS := os.DirFS("../../config")
	err := ReadRoleDir(nerthusFS, "builtin_roles", roles)
	if err != nil {
		t.Error(err)
		return
	}
	buriRole, ok := roles["buri"]
	if !ok {
		t.Error("missing buri role")
		return
	}
	if buriRole.Name == "" {
		t.Error("buri role is missing a name")
		return
	}
	if len(buriRole.Tasks) == 0 {
		t.Error("buri role is missing tasks")
		return
	}
	if len(buriRole.Vars) == 0 {
		t.Error("buri role is missing var definitions")
		return
	}
	if len(buriRole.Dependencies) == 0 && buriRole.Dependencies[0].Role != "user" {
		t.Error("buri role is missing user as first dependencie")
		return
	}
}

func TestDirDoesNotExistNoError(t *testing.T) {
	roles := make(map[string]Role)
	noFS := os.DirFS("DoesNotExist")
	err := ReadRoleDir(noFS, "roles", roles)
	if err != nil {
		t.Error(err)
		return
	}
}
