package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/10/pkg/tweeter"
)

var (
	message, apiKey, apiKeySecret, accessToken, accessTokenSecret string
	dryRun, versionFlag                                           bool
)

func main() {
	parseAndValidateInput()

	if versionFlag {
		printVersion()
		return
	}

	if dryRun {
		printOutput("sentMessage", message)
		return
	}

	tweeterClient, err := tweeter.New(tweeter.Config{
		ApiKey:            apiKey,
		ApiKeySecret:      apiKeySecret,
		AccessToken:       accessToken,
		AccessTokenSecret: accessTokenSecret,
	})

	if err != nil {
		err = errors.Wrap(err, "failed creating tweeter client")
		printOutput("errorMessage", err.Error())
		os.Exit(1)
	}

	// send the tweet
	if err := tweeterClient.Tweet(message); err != nil {
		err = errors.Wrap(err, "status update error")
		printOutput("errorMessage", err.Error())
		os.Exit(1)
	}

	printOutput("sentMessage", message)
}

func parseAndValidateInput() {
	flag.StringVar(&message, "message", "", "message you'd like to send to twitter")
	flag.StringVar(&apiKey, "apiKey", "", "twitter api key")
	flag.StringVar(&apiKeySecret, "apiKeySecret", "", "twitter api key secret")
	flag.StringVar(&accessToken, "accessToken", "", "twitter access token")
	flag.StringVar(&accessTokenSecret, "accessTokenSecret", "", "twitter access token secret")
	flag.BoolVar(&dryRun, "dryRun", false, "if true or if env var DRY_RUN=true, then a tweet will not be sent")
	flag.BoolVar(&versionFlag, "version", false, "output the version of tweeter")
	flag.Parse()

	if os.Getenv("DRY_RUN") == "true" {
		dryRun = true
	}

	if versionFlag {
		return
	}

	var err error
	if message == "" {
		err = multierror.Append(err, errors.New("--message can't be empty"))
	}

	if !dryRun {
		if apiKey == "" {
			err = multierror.Append(err, errors.New("--apiKey can't be empty"))
		}

		if apiKeySecret == "" {
			err = multierror.Append(err, errors.New("--apiKeySecret can't be empty"))
		}

		if accessToken == "" {
			err = multierror.Append(err, errors.New("--accessToken can't be empty"))
		}

		if accessTokenSecret == "" {
			err = multierror.Append(err, errors.New("--accessTokenSecret can't be empty"))
		}
	}

	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func printVersion() {
	versionStr := "dirty"
	if tweeter.Version != "" {
		versionStr = tweeter.Version
	}
	fmt.Printf("tweeter version: %s", versionStr)
}

func printOutput(key, message string) {
	fmt.Printf("::set-output name=%s::%s\n", key, message)
}
