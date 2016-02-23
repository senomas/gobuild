package main

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

type fexec func(name string, path, param []string)

var env []string

var gparams = make(map[string]string)

func main() {
	var err error
	var data []byte

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Println("ERROR: ", r)
	// 	}
	// }()

	for _, v := range os.Args[1:] {
		if v == "-u" {
			gparams["update"] = "true"
		}
	}

	var cd string
	if cd, err = filepath.Abs("."); err != nil {
		panic(err)
	}
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GOPATH=") {
			env = append(env, v)
		}
	}
	env = append(env, fmt.Sprintf("GOPATH=%s", cd))

	if data, err = ioutil.ReadFile("gobuild.yaml"); err != nil {
		panic(fmt.Sprintf("Error reading config\n%+v\n\n", err))
	}

	var cfg map[interface{}]interface{}
	cfg = make(map[interface{}]interface{})

	if err = yaml.Unmarshal(data, &cfg); err != nil {
		panic(fmt.Sprintf("Error parsing config\n\n%+v\n", err))
	}

	process1(cfg, "tool")
	process1(cfg, "lib")
	process1(cfg, "src-pre-build")
	process1(cfg, "build")
	process1(cfg, "install")
	process1(cfg, "test")

	fmt.Printf("\n\n\n====== DONE GOBUILD ================================\n\n\n")
}

func process1(cfg map[interface{}]interface{}, name string) {
	value, ok := cfg[name]
	if ok {
		fmt.Printf("\n\n\n====== RUNNING %s ================================\n", strings.ToUpper(name))
		process([]string{name}, value)
	}
}

func process(path []string, value interface{}) {
	var str string
	var ok bool
	if str, ok = value.(string); ok {
		runExec(path, []string{str})
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
			runExec(path, ss)
		} else if tt == "map" {
			mv := value.(map[interface{}]interface{})
			for mk, mv := range mv {
				if str, ok = mk.(string); ok {
					npath := append(path, str)
					process(npath, mv)
				} else {
					panic(fmt.Sprintf("Error param [%+v] = [%+v]\n", mk, mv))
				}
			}
		} else {
			panic(fmt.Sprintf("Not supported %s\n%+v\n", tt, value))
		}
	}
}

func runExec(path, params []string) {
	if path[0] == "src-pre-build" {
		var fp bytes.Buffer
		fp.WriteString("src")
		for _, v := range path[1 : len(path)-1] {
			fp.WriteRune(os.PathSeparator)
			fp.WriteString(v)
		}
		var ps []string
		for _, v := range regSplit(params[0], "\\s+") {
			if v == "{}" {
				ps = concat(ps, scan(fp.String(), path[len(path)-1]))
			} else {
				ps = append(ps, v)
			}
		}
		osExec(path[1], ps)
	} else {
		var ps = []string{"go"}
		p0 := path[0]
		if p0 == "lib" || p0 == "tool" {
			ps = concat(ps, []string{"get"})
		} else {
			ps = append(ps, p0)
		}
		if gparams["update"] == "true" {
			ps = append(ps, "-u")
		}
		ps = concat(ps, params)
		osExec(path[1], ps)
	}
}

func osExec(name string, params []string) {
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

func concat(p1, p2 []string) []string {
	for _, v := range p2 {
		p1 = append(p1, v)
	}
	return p1
}

func scan(path string, pattern string) []string {
	var files []string
	doScan(&files, path, pattern)
	return files
}

func doScan(files *[]string, path string, pattern string) {
	var fz []os.FileInfo
	var err error
	if fz, err = ioutil.ReadDir(path); err != nil {
		panic(err)
	}
	for _, f := range fz {
		var fn string
		if path == "" {
			fn = f.Name()
		} else {
			fn = path + "/" + f.Name()
		}
		if f.IsDir() {
			doScan(files, path+"/"+f.Name(), pattern)
		} else {
			var b bool
			if b, err = filepath.Match(pattern, f.Name()); err != nil {
				panic(err)
			}
			if b {
				*files = append(*files, fn)
			}
		}
	}
}

func regSplit(text string, delimeter string) []string {
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
