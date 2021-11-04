#!/usr/bin/env bash

set -x
set -e

CGO_ENABLED=0 go build -gcflags "all=-N -l" -o /tmp/app

#dlv --listen=:2345 --headless=true --log=true --log-output=debugger,debuglineerr,gdbwire,lldbout,rpc --accept-multiclient --api-version=2 exec /tmp/app
dlv --listen=:2345 --headless=true --accept-multiclient --api-version=2 exec /tmp/app
