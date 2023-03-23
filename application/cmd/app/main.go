package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	buffalo "github.com/gobuffalo/buffalo/runtime"
	"github.com/gobuffalo/buffalo/servers"

	"github.com/silinternational/cover-api/actions"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
)

// main is the starting point for your Buffalo application.
// You can feel free and add to this `main` method, change
// what it does, etc...
// All we ask is that, at some point, you make sure to
// call `app.Serve()`, unless you don't want to start your
// application that is. :)
func main() {
	srv, err := getServer()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	app := actions.App()

	log.Infof("Go version: %s", runtime.Version())
	log.Infof("Buffalo version: %s", buffalo.Version)
	log.Infof("Buffalo build info: %s", buffalo.Build())
	log.Infof("Commit hash: %s", strings.TrimSpace(domain.Commit))

	if err := app.Serve(srv); err != nil {
		if err.Error() != "context canceled" {
			panic(err)
		}
		os.Exit(0)
	}
}

func getServer() (servers.Server, error) {
	const (
		certFile = "cert.pem"
		keyFile  = "key.pem"
	)

	if domain.Env.DisableTLS {
		return servers.New(), nil
	}

	err := generateCert(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("generate cert: %w", err)
	}

	cfg, err := tlsConfig(certFile, keyFile)
	if err != nil {
		return servers.New(), fmt.Errorf("get TLS config: %w", err)
	}
	listener, err := tls.Listen("tcp", fmt.Sprintf(":%d", domain.Env.Port), cfg)
	if err != nil {
		return servers.New(), fmt.Errorf("get TLS listener: %w", err)
	}

	return servers.WrapListener(&http.Server{ReadHeaderTimeout: time.Second * 15}, listener), nil
}

func tlsConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load cert/key files: %w", err)
	}

	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	return &config, nil
}

/*
# Notes about `main.go`

## SSL Support

We recommend placing your application behind a proxy, such as
Apache or Nginx and letting them do the SSL heavy lifting
for you. https://gobuffalo.io/en/docs/proxy

## Buffalo Build

When `buffalo build` is run to compile your binary, this `main`
function will be at the heart of that binary. It is expected
that your `main` function will start your application using
the `app.Serve()` method.

*/
