#!/bin/bash
if [[ ! -d bin ]]; then
  mkdir bin
fi
go build -o bin/gophette
go get -u github.com/gonutz/payload/cmd/payload
payload -exe=bin/gophette -data=resource/resources.blob
