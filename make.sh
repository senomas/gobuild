#!/bin/bash
GOPATH=/Users/seno/workspace/gobuild
go install -v senomas/gobuild senomas/gox
cp bin/gobuild /usr/local/bin/
cp bin/gox /usr/local/bin/
