package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/gobuffalo/buffalo/servers"
	"github.com/libdns/cloudflare"
	"github.com/rollbar/rollbar-go"

	dynamodbstore "github.com/silinternational/certmagic-storage-dynamodb"

	"github.com/silinternational/cover-api/actions"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/job"
)

var GitCommitHash string

// main is the starting point for your Buffalo application.
// You can feel free and add to this `main` method, change
// what it does, etc...
// All we ask is that, at some point, you make sure to
// call `app.Serve()`, unless you don't want to start your
// application that is. :)
func main() {
	delay := time.Duration(time.Second * 10)

	// Kick off first run of inactivating items between 1h11 and 3h27 from now
	if domain.Env.GoEnv != "development" {
		randMins := time.Duration(domain.RandomInsecureIntInRange(71, 387))
		delay = randMins * time.Minute
	}

	if err := job.SubmitDelayed(job.InactivateItems, delay, map[string]interface{}{}); err != nil {
		domain.ErrLogger.Printf("error initializing InactivateItems job: " + err.Error())
		os.Exit(1)
	}

	// init rollbar
	rollbar.SetToken(domain.Env.RollbarToken)
	rollbar.SetEnvironment(domain.Env.GoEnv)
	rollbar.SetCodeVersion(GitCommitHash)
	rollbar.SetServerRoot(domain.Env.RollbarServerRoot)

	srv, err := getServer()
	if err != nil {
		domain.ErrLogger.Printf(err.Error())
		os.Exit(1)
	}

	app := actions.App()
	rollbar.WrapAndWait(func() {
		if err := app.Serve(srv); err != nil {
			if err.Error() != "context canceled" {
				panic(err)
			}
			os.Exit(0)
		}
	})
}

func getServer() (servers.Server, error) {
	if domain.Env.DisableTLS {
		return servers.New(), nil
	}

	certmagic.Default.Storage = &dynamodbstore.Storage{
		Table:     domain.Env.DynamoDBTable,
		AwsRegion: domain.Env.AwsRegion,
	}

	if !domain.IsProduction() {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}

	certmagic.DefaultACME.Email = domain.Env.SupportEmail
	certmagic.DefaultACME.Agreed = true
	certmagic.HTTPSPort = domain.Env.ServerPort
	certmagic.Default.DefaultServerName = domain.Env.CertDomainName
	certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
		DNSProvider: &cloudflare.Provider{
			APIToken: domain.Env.CloudflareToken,
		},
	}

	listener, err := certmagic.Listen([]string{domain.Env.CertDomainName})
	if err != nil {
		return servers.New(), fmt.Errorf("failed to get TLS config: %s", err.Error())
	}

	return servers.WrapListener(&http.Server{}, listener), nil
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
