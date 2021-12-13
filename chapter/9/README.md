# DevOps for Go Tweeter
The tweeter command line tool will send a tweet via Twitter.

## Setup
You can use tweeter to send a tweet or to output the message to STDOUT. If you want to send a tweet, you will need to set up a Twitter application.

### Setup With a Twitter Application
To send a tweet, you will need to create or use an existing Twitter account, create a Twitter application, and generate API credentials. All of this can be done through the
[Twitter Developer Portal](https://developer.twitter.com/en/portal/projects-and-apps).

### Setup Without a Twitter Application
Some people may not want to set up a Twitter account. If you would like to use tweeter without sending tweets, use the `--dry-run` argument. This will cause the tool to write the message to STDOUT rather than sending the message to Twitter.

## Inputs

- `--message` **Required** the tweet message you would like to send
- `--apiKey` the API key under Consumer Keys in the [Twitter developer portal](https://developer.twitter.com/en/portal/projects-and-apps)
- `--apiKeySecret` the API key secret under Consumer Keys in the [Twitter developer portal](https://developer.twitter.com/en/portal/projects-and-apps)
- `--accessToken` the access token under Authentication Tokens in the [Twitter developer portal](https://developer.twitter.com/en/portal/projects-and-apps)
- `--accessTokenSecret` the access token secret under Authentications Tokens in the [Twitter developer portal](https://developer.twitter.com/en/portal/projects-and-apps)
- `--dryRun` will skip authentication validation and sending the message to Twitter

## Test
```
$ go test ./...
?   	github.com/devopsforgo/github-actions	[no test files]
ok  	github.com/devopsforgo/github-actions/pkg/tweeter	0.002s
```

## Run Help
To see the command line arguments and descriptions, display the help.
```
$ go run . -h
Usage of /tmp/go-build3731631588/b001/exe/github-actions:
      --accessToken string         twitter access token
      --accessTokenSecret string   twitter access token secret
      --apiKey string              twitter api key
      --apiKeySecret string        twitter api key secret
      --dryRun                     if true, then a tweet will not be sent
      --message string             message you'd like to send to twitter
pflag: help requested
exit status 2
```

## Run Without Sending a Tweet
The `--dryRun` argument will skip validation of the authentication arguments and output the message to STDOUT.
```
$ go run . --dryRun --message foo
```

## Run Sending a Tweet
Without `--dryRun` specified, tweeter will send the `--messsage` argument as a Tweet.
```
$ go run . --message foo --apiKey 123 --apiKeySecret secret --accessToken token --accessTokenSecret secret
```