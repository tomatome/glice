package mod

import (
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const goMod = "go.mod"

func Exists(path string) bool {
	if _, err := os.Stat(filepath.Join(path, goMod)); err == nil || os.IsExist(err) {
		return true
	}
	return false
}

func Parse(path string, withIndirect bool) ([]module.Version, error) {
	bts, err := os.ReadFile(filepath.Join(path, goMod))
	if err != nil {
		return nil, err
	}

	modFile, err := modfile.Parse("go.mod", bts, nil)
	if err != nil {
		return nil, err
	}

	var deps []module.Version
	for _, f := range modFile.Require {
		if f.Indirect && !withIndirect {
			continue
		}
		deps = append(deps, f.Mod)
	}

	return deps, nil
}
