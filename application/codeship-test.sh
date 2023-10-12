#!/usr/bin/env bash

# output everything
set -e
# exit on first error
set -x

#curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $GOPATH/bin latest
go install github.com/securego/gosec/v2/cmd/gosec@v2.17.0

# https://github.com/securego/gosec#available-rules
# G104 ignore errors not checked
gosec -exclude=G104 -quiet ./...

whenavail testdb 5432 10 buffalo-pop pop migrate up
buffalo test
