package gocurse

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type PkgMeta struct {
	PackageAs       string   `yaml:"package-as"`
	ManualChangelog string   `yaml:"manual-changelog"`
	Ignore          []string `yaml:"ignore,flow"`
}

func GetPkgMeta(path ...string) (*PkgMeta, error) {
	if len(path) == 0 {
		path = []string{".pkgmeta"}
	}

	fp, err := os.Open(path[0])
	if err != nil {
		return nil, nil
	}
	defer fp.Close()

	data, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, err
	}

	var meta PkgMeta
	err = yaml.Unmarshal(data, &meta)
	if err != nil {
		return nil, err
	}

	return &meta, nil
}
