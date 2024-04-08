package main

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	buffalo "github.com/gobuffalo/buffalo/runtime"

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
	app := actions.App()

	log.Info("Go version:", runtime.Version())
	log.Info("Buffalo version:", buffalo.Version)
	log.Info("Buffalo build info:", buffalo.Build())
	log.Info("Commit hash:", strings.TrimSpace(domain.Commit))

	const (
		certFile = "cert.pem"
		keyFile  = "key.pem"
	)

	if err := generateCert(certFile, keyFile); err != nil {
		log.Fatalf("failed to generate cert: %s", err)
	}

	if err := app.StartTLS(fmt.Sprintf(":%d", domain.Env.Port), certFile, keyFile); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}
