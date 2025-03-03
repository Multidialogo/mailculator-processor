#!/bin/sh

go test ./...
chown 1000:1000 -R .
