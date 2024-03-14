FROM golang:1.22 as builder

ENV REFRESHED_AT=2024-03-13

ARG GITHUB_SHA

ENV GOPROXY=http://proxy.golang.org

WORKDIR /app

# Install buffalo
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY . .

RUN sh -c "echo $GITHUB_SHA > domain/commit.txt"
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
