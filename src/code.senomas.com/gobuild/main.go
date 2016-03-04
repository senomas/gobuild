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
	"syscall"
	"unicode"
)

type fexec func(name string, path, param []string)

var (
	env     []string
	gparams = map[string][]string{
		"build": []string{},
		"get":   []string{},
		"run":   []string{},
		"test":  []string{},
	}
)

func main() {
	var err error
	var data []byte

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Println("ERROR: ", r)
	// 	}
	// }()

	var base = "build"
	for _, v := range os.Args[1:] {
		switch v {
		default:
			gparams[base] = append(gparams[base], v)
		case "--build":
			base = "build"
		case "--get":
			base = "get"
		case "--run":
			base = "run"
		case "--test":
			base = "test"
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
	process1(cfg, "exec")

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
	} else if path[0] == "exec" {
		if len(params) == 1 {
			var xp []string
			var buf bytes.Buffer
			var state = 0
			for _, c := range params[0] {
				switch state {
				case 0:
					if !unicode.IsSpace(c) {
						buf.WriteRune(c)
						state = 1
					}
				case 1:
					if unicode.IsSpace(c) {
						xp = append(xp, buf.String())
						buf.Reset()
						state = 0
					} else if c == '\'' {
						buf.WriteRune(c)
						state = 10
					} else if c == '"' {
						buf.WriteRune(c)
						state = 20
					} else {
						buf.WriteRune(c)
					}
				case 10:
					if c == '\'' {
						buf.WriteRune(c)
						state = 1
					} else {
						buf.WriteRune(c)
					}
				case 20:
					if c == '"' {
						buf.WriteRune(c)
						state = 1
					} else {
						buf.WriteRune(c)
					}
				}
			}
			if state == 1 {
				xp = append(xp, buf.String())
			}
			osExec(path[1], xp)
		} else {
			osExec(path[1], params)
		}
	} else {
		var ps = []string{"go"}
		p0 := path[0]
		if p0 == "lib" || p0 == "tool" {
			ps = concat(ps, []string{"get"})
			ps = concat(ps, gparams["get"])
		} else {
			ps = append(ps, p0)
			ps = concat(ps, gparams[p0])
		}
		ps = concat(ps, params)
		osExec(path[1], ps)
	}
}

func osExec(name string, params []string) {
	var err error
	fmt.Printf("%s:\n%s\n", name, flatten(params))
	var cmd *exec.Cmd
	if len(params) == 1 {
		cmd = exec.Command(params[0])
	} else {
		cmd = exec.Command(params[0], params[1:]...)
	}
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		panic(err)
	}
	if err = cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
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
