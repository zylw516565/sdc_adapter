#!/bin/bash
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=arm64
export GOARM=7
go build -o demoArm