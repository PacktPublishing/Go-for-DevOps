package tweeter

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

var (
	// Version is the git reference injected at build
	Version string
)

type (
	// Config is the authentication params needed to construct a tweeter.Client
	Config struct {
		ApiKey            string
		ApiKeySecret      string
		AccessToken       string
		AccessTokenSecret string
	}

	// Client sends tweets to Twitter
	Client struct {
		twitterClient *twitter.Client
	}
)

// Validate will check the Config to ensure its field values are valid
func (cfg Config) Validate() error {
	var err error
	if cfg.ApiKey == "" {
		err = multierror.Append(err, errors.New("ApiKey is required"))
	}

	if cfg.ApiKeySecret == "" {
		err = multierror.Append(err, errors.New("ApiKeySecret is required"))
	}

	if cfg.AccessToken == "" {
		err = multierror.Append(err, errors.New("AccessToken is required"))
	}

	if cfg.AccessTokenSecret == "" {
		err = multierror.Append(err, errors.New("AccessTokenSecret is required"))
	}
	return err
}

// New creates a new instance of the tweeter.Client ready to send tweets
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate tweeter config")
	}

	var (
		oauthCfg   = oauth1.NewConfig(cfg.ApiKey, cfg.ApiKeySecret)
		token      = oauth1.NewToken(cfg.AccessToken, cfg.AccessTokenSecret)
		httpClient = oauthCfg.Client(oauth1.NoContext, token)
	)

	return &Client{
		twitterClient: twitter.NewClient(httpClient),
	}, nil
}

// Tweet sends a tweet
func (c *Client) Tweet(message string) error {
	_, _, err := c.twitterClient.Statuses.Update(message, nil)
	return errors.Wrap(err, "failed to send tweet")
}
