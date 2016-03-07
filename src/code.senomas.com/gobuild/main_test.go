package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

func TestParseYaml(t *testing.T) {
	var err error
	var data []byte

	if data, err = ioutil.ReadFile("/Users/seno/workspace/picloud/gobuild.yaml"); err != nil {
		t.Fatalf("Error reading config\n%+v\n\n", err)
	}

	if len(data) == 0 {
		t.Fail()
	}

	var cfg yaml.MapSlice

	if err = yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Error parsing config\n\n%+v\n", err)
	}

	t.Logf("Data %+v\n", cfg)
}
