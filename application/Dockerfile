# This is a multi-stage Dockerfile and requires >= Docker 17.05
# https://docs.docker.com/engine/userguide/eng-image/multistage-build/
FROM gobuffalo/buffalo:v0.18.8 as builder

ARG ci_commit_id

WORKDIR /src

ENV GOPROXY http://proxy.golang.org

# Install the desired version of NodeJS
RUN npm cache clean -f && \
    npm install -g n && \
    n 16

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY . .

RUN sh -c "echo $ci_commit_id > domain/commit.txt"
RUN cat domain/commit.txt

RUN buffalo build --static -o /bin/app

FROM alpine:3
RUN apk add --no-cache bash ca-certificates

WORKDIR /bin/

COPY --from=builder /bin/app .

# Uncomment to run the binary in "production" mode:
ENV GO_ENV=production

# Bind the app to 0.0.0.0 so it can be seen from outside the container
ENV ADDR=0.0.0.0

EXPOSE 3000

CMD /bin/app migrate && /bin/app
