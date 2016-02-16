package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
)

type fexec func(name string, param []string)

var env []string

func main() {
	var err error
	var data []byte

	var cd string
	if cd, err = filepath.Abs("."); err != nil {
		panic(err)
	}
	env = append(os.Environ(), fmt.Sprintf("GOPATH=%s", cd))

	if data, err = ioutil.ReadFile("build.yaml"); err != nil {
		fmt.Printf("Error reading config\n%+v\n\n", err)
		os.Exit(1)
	}

	var cfg map[interface{}]interface{}
	cfg = make(map[interface{}]interface{})

	if err = yaml.Unmarshal(data, &cfg); err != nil {
		panic(fmt.Sprintf("Error parsing config\n\n%+v\n", err))
	}

	// fmt.Printf("CONFIG\n%+v\n", cfg)

	buildTask(cfg, "tool", goGet)

	buildTask(cfg, "lib", goGet)

	buildTask(cfg, "src-pre-build", goSrcPreBuild)

	buildTask(cfg, "install", goInstall)
}

func buildTask(cfg map[interface{}]interface{}, name string, fn fexec) {
	task, ok := cfg[name]
	if ok {
		mv := task.(map[interface{}]interface{})
		for k, v := range mv {
			if str, ok := k.(string); ok {
				process(fn, str, v)
			} else {
				panic(fmt.Sprintf("Error param %+v\n", k))
			}
		}
	}
}

func buildExec(name string, params []string) {
	var err error
	fmt.Printf("%s:\n%s\n", name, flatten(params))
	cmd := exec.Command(params[0], params[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		os.Exit(1)
	}
	fmt.Println()
}

func goGet(name string, params []string) {
	buildExec(name, concat([]string{"go", "get", "-v"}, params))
}

func goBuild(name string, params []string) {
	buildExec(name, concat([]string{"go", "build", "-v"}, params))
}

func goInstall(name string, params []string) {
	buildExec(name, concat([]string{"go", "install", "-v"}, params))
}

func goSrcPreBuild(name string, params []string) {
	var err error
	if len(params) != 1 {
		panic(fmt.Sprintf("Invalid params length %+v", params))
	}
	filter := filterSrc(regexp.MustCompile(name))
	var prg string
	var gc []string
	for k, v := range RegSplit(params[0], "\\s+") {
		if k == 0 {
			prg = v
		} else if v == "{}" {
			gc = concat(gc, filter)
		} else {
			gc = append(gc, v)
		}
	}
	fmt.Printf("%s:\n%s %s\n", name, prg, flatten(gc))
	cmd := exec.Command(prg, gc...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		os.Exit(1)
	}
	fmt.Println()
}

func process(fn fexec, k string, value interface{}) {
	var str string
	var ok bool
	if str, ok = value.(string); ok {
		fn(k, []string{str})
	} else {
		tt := reflect.TypeOf(value).Kind().String()
		if tt == "slice" {
			var ss []string
			mv := value.([]interface{})
			for _, v := range mv {
				if str, ok = v.(string); ok {
					ss = append(ss, str)
				} else {
					panic(fmt.Sprintf("Error param %+v\n", value))
				}
			}
			fn(k, ss)
		} else {
			fmt.Printf("Not supported %s\n%+v\n", tt, value)
			os.Exit(1)
		}
	}
}

func concat(p1, p2 []string) []string {
	for _, v := range p2 {
		p1 = append(p1, v)
	}
	return p1
}

func filterSrc(filter *regexp.Regexp) []string {
	var files []string
	filterSrcRec(&files, "src", "src", filter)
	return files
}

func filterSrcRec(files *[]string, path, name string, filter *regexp.Regexp) {
	var fz []os.FileInfo
	var err error
	if fz, err = ioutil.ReadDir(path); err != nil {
		panic(err)
	}
	for _, f := range fz {
		var fn string
		if name == "" {
			fn = f.Name()
		} else {
			fn = name + "/" + f.Name()
		}
		if f.IsDir() {
			filterSrcRec(files, path+"/"+f.Name(), fn, filter)
		} else {
			if filter.MatchString(fn) {
				*files = append(*files, fn)
			}
		}
	}
}

func RegSplit(text string, delimeter string) []string {
	reg := regexp.MustCompile(delimeter)
	indexes := reg.FindAllStringIndex(text, -1)
	laststart := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[laststart:element[0]]
		laststart = element[1]
	}
	result[len(indexes)] = text[laststart:len(text)]
	return result
}

func flatten(ss []string) string {
	var s string
	for k, v := range ss {
		if k != 0 {
			s = s + " " + v
		} else {
			s = v
		}
	}
	return s
}
