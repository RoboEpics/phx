package pei

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"

	"gopkg.in/yaml.v3"
)

type PEI map[string]PlatformExecution

func (pei PEI) Do(req peiRequest) (output []byte, err error) {
	pe, ok := pei[req.script()]
	if !ok {
		return nil, fmt.Errorf("script %s not defined", req.script())
	}
	return pe.Exec(req.env())
}

func LoadPEI(dir string) (PEI, error) {
	type decoder = func(in []byte, out any) error
	load := func(filename string, unmarshal decoder) (PEI, error) {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		buf, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		var pei PEI
		if err := unmarshal(buf, &pei); err != nil {
			return nil, err
		}
		return pei, nil
	}
	nameDecoder := map[string]decoder{
		"pei.yaml": yaml.Unmarshal,
		"pei.yml":  yaml.Unmarshal,
		"pei.json": json.Unmarshal,
	}
	for n, d := range nameDecoder {
		peiPath := path.Join(dir, n)
		pei, err := load(peiPath, d)
		if err == nil {
			return pei, nil
		}
	}
	return nil, fmt.Errorf("couldnot load pei files")
}

type PlatformExecution struct {
	Linux   Execution `json:"linux" yaml:"linux"`
	Darwin  Execution `json:"darwin" yaml:"darwin"`
	Windows Execution `json:"windows" yaml:"windows"`
}

func (pe PlatformExecution) _map() map[string]Execution {
	return map[string]Execution{
		"linux":   pe.Linux,
		"darwin":  pe.Darwin,
		"windows": pe.Windows,
	}
}

type Execution struct {
	CMD  string   `json:"cmd" yaml:"cmd"`
	Args []string `json:"args" yaml:"args"`
}

func (pe PlatformExecution) Exec(env map[string]string) ([]byte, error) {
	var (
		goos = runtime.GOOS
	)
	e, ok := pe._map()[goos]
	if !ok || e.CMD == "" {
		return nil, fmt.Errorf("unsupported platform %s", goos)
	}
	cmd := exec.Command(e.CMD, e.Args...)
	for k, v := range env {
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("%s=%s", k, v))
	}
	return cmd.Output()
}
