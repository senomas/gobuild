#!/bin/bash
GOPATH=/Users/seno/workspace/gobuild
go install -v code.senomas.com/gobuild code.senomas.com/gox
cp bin/gobuild /usr/local/bin/
cp bin/gox /usr/local/bin/
