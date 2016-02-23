package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	var env []string
	var err error

	f, err := os.OpenFile("/Users/seno/Temp/gox.log", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	for _, s := range os.Args {
		if _, err = f.WriteString(s); err != nil {
			panic(err)
		}
		if _, err = f.WriteString(" "); err != nil {
			panic(err)
		}
	}
	if _, err = f.WriteString("\n"); err != nil {
		panic(err)
	}

	var path = baseDir()

	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GOPATH=") {
			env = append(env, v)
		}
	}
	env = append(env, fmt.Sprintf("GOPATH=%s", path))

	cmd := exec.Command("go", os.Args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		os.Exit(1)
	}
	fmt.Println()
}

func baseDir() string {
	var err error
	var path string
	if path, err = filepath.Abs("."); err != nil {
		panic(err)
	}
	return baseDirRec(path)
}

func baseDirRec(path string) string {
	var err error
	if _, err = os.Stat(path + "/gobuild.yaml"); os.IsNotExist(err) {
		path = filepath.Dir(path)
		if "." == path {
			panic("No gobuild.yaml found!")
		}
		return baseDirRec(path)
	}
	return path
}
