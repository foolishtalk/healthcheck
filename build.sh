#!/bin/sh
go build -tags netgo -ldflags '-s -w' -o healthcheck_app
