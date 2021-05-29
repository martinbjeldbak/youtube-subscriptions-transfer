# YouTube Subscriptions Transfer Script

This Go script uses the YouTube API to transfer subscriptions from one YouTube account to another. Login access to both accounts and a Google Cloud account are required.

By transfer, I mean it will make YouTube account `B` (target account) subscribe to the exact same channels as in YouTube account `A` (source account). There is no loss of data.

The script is heavily based on the YouTube Go quickstart tutorial [here](https://developers.google.com/youtube/v3/quickstart/go), which requires a Google Cloud account. Follow the setup of this tutorial before running this script and you should be good to go.

Background: I needed to create a new YouTube account to join a family subscription and had 1,000+ subscriptions on my existing account. I was too lazy to manually subscribe to each content creator.

## Running

Prerequisites: Golang >= 1.16 and a Google Cloud account with your API secret created in the quickstart tutorial saved in `client_secret.json`.

When running the below commands, dependencies will be downloaded and you will be presented links to authenticate both source and target YouTube accounts using OAuth and paste in the access token into the terminal.

```sh
go mod download
go run main.go
```

Once this is done, the transfer process will start. See note below for caveats.

Note: Due to [quota limits](https://developers.google.com/youtube/v3/determine_quota_cost#subscriptions) on the YouTube API, you may need to run this once every day for multiple days to transfer hundreds to thousands of subscriptions.

To keep track of state, an `importStatus.gob` file is created. __Do not__ delete this file if you are hitting quota limits.

Once everything has been transferred, you can remove all files.

## Contributing

Discovered a bug or got stuck? Please create a new issue in the repository and assign it to me and I will do my best to address.