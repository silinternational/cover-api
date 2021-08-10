#!/usr/bin/env ash

# output everything
set -e
# exit on first error
set -x

# generate swagger.json
swagger generate spec -m -o /src/swagger.json

# install redoc-cli to bundle html
apk add --no-cache npm
npm install -g redoc-cli
redoc-cli bundle --cdn --title "Riskman API Documentation" --output api-docs.html /src/swagger.json

# install aws cli to push docs to s3
apk add --no-cache python3 py3-pip
pip install awscli

# copy file to s3
aws s3 cp --acl public-read api-docs.html $S3PATH
