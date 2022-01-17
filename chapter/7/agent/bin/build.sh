#!/bin/sh

cd ..
env GOOS=linux GOARCH=amd64 go build -o bin/linux_amd64/agent
