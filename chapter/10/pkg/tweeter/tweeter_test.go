package tweeter_test

import (
	"strings"
	"testing"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/10/pkg/tweeter"
)

func TestNew(t *testing.T) {
	subject, err := tweeter.New(tweeter.Config{
		ApiKey:            "apiKey",
		ApiKeySecret:      "apiKeySecret",
		AccessToken:       "accessToken",
		AccessTokenSecret: "accessTokenSecret",
	})

	if err != nil {
		t.Error(err, "config should be valid")
	}

	if subject == nil {
		t.Error("subject should not be nil")
	}
}

func TestConfig_Validate(t *testing.T) {
	testCases := []struct {
		Name   string
		Cfg    tweeter.Config
		Expect func(t *testing.T, err error)
	}{
		{
			Name: "All keys are filled",
			Cfg: tweeter.Config{
				ApiKey:            "apiKey",
				ApiKeySecret:      "apiKeySecret",
				AccessToken:       "accessToken",
				AccessTokenSecret: "accessTokenSecret",
			},
			Expect: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err, "error should be nil")
				}
			},
		},
		{
			Name: "ApiKey is empty",
			Cfg: tweeter.Config{
				ApiKey:            "",
				ApiKeySecret:      "apiKeySecret",
				AccessToken:       "accessToken",
				AccessTokenSecret: "accessTokenSecret",
			},
			Expect: func(t *testing.T, err error) {
				if err == nil {
					t.Error("error should be non-nil")
				}

				if !strings.Contains(err.Error(), "ApiKey is required") {
					t.Error(err.Error(), "should contain 'ApiKey is required'")
				}
			},
		},
		{
			Name: "ApiKeySecret is empty",
			Cfg: tweeter.Config{
				ApiKey:            "apiKey",
				ApiKeySecret:      "",
				AccessToken:       "accessToken",
				AccessTokenSecret: "accessTokenSecret",
			},
			Expect: func(t *testing.T, err error) {
				if err == nil {
					t.Error("error should be non-nil")
				}

				if !strings.Contains(err.Error(), "ApiKeySecret is required") {
					t.Error(err.Error(), "should contain 'ApiKeySecret is required'")
				}
			},
		},
		{
			Name: "AccessToken is empty",
			Cfg: tweeter.Config{
				ApiKey:            "apiKey",
				ApiKeySecret:      "apiKeySecret",
				AccessToken:       "",
				AccessTokenSecret: "accessTokenSecret",
			},
			Expect: func(t *testing.T, err error) {
				if err == nil {
					t.Error("error should be non-nil")
				}

				if !strings.Contains(err.Error(), "AccessToken is required") {
					t.Error(err.Error(), "should contain 'AccessToken is required'")
				}
			},
		},
		{
			Name: "AccessTokenSecret is empty",
			Cfg: tweeter.Config{
				ApiKey:            "apiKey",
				ApiKeySecret:      "apiKeySecret",
				AccessToken:       "accessToken",
				AccessTokenSecret: "",
			},
			Expect: func(t *testing.T, err error) {
				if err == nil {
					t.Error("error should be non-nil")
				}

				if !strings.Contains(err.Error(), "AccessTokenSecret is required") {
					t.Error(err.Error(), "should contain 'AccessTokenSecret is required'")
				}
			},
		},
		{
			Name: "ApiKey and AccessTokenSecret are empty",
			Cfg: tweeter.Config{
				ApiKey:            "",
				ApiKeySecret:      "apiKeySecret",
				AccessToken:       "accessToken",
				AccessTokenSecret: "",
			},
			Expect: func(t *testing.T, err error) {
				if err == nil {
					t.Error("error should be non-nil")
				}

				if !strings.Contains(err.Error(), "AccessTokenSecret is required") ||
					!strings.Contains(err.Error(), "ApiKey is required") {
					t.Error(err.Error(), "should contain 'AccessTokenSecret is required' and 'ApiKey is required'")
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			tc.Expect(t, tc.Cfg.Validate())
		})
	}
}
