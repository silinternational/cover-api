#!/usr/bin/env ash

# output everything
set -e
# exit on first error
set -x

# generate swagger.json
swagger generate spec -m -o /src/swagger.json

# bundle doc html
redoc-cli bundle --cdn --title "Cover API Documentation" --output api-docs.html /src/swagger.json

# copy file to s3
aws s3 cp api-docs.html $S3PATH
