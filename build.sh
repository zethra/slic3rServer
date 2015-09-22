#!/bin/bash
export GOPATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)
go build -o $GOPATH/bin/slic3rServer $GOPATH/src/main.go