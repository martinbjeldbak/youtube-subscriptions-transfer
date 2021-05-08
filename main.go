package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}

func mySubscriptions(context context.Context, service *youtube.Service, parts []string) ([]*youtube.Subscription, error) {
	call := service.Subscriptions.List(parts)
	call.Mine(true)

	var channels = make([]*youtube.Subscription, 0)

	err := call.Pages(context, func(slr *youtube.SubscriptionListResponse) error {
		channels = append(channels, slr.Items...)

		return nil
	})
	return channels, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(ctx context.Context, config *oauth2.Config, name string) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf(name+" account: Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile(name string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape(name+".json")), err
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config, name string) *http.Client {
	cacheFile, err := tokenCacheFile(name)
	if err != nil {
		log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(ctx, config, name)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

type ChannelImportStatus struct {
	Channel  *youtube.Subscription
	Imported bool
}

func getService(ctx context.Context, kind string, clientSecret []byte) *youtube.Service {
	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/kind.json
	config, err := google.ConfigFromJSON(clientSecret, youtube.YoutubeReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config, kind)

	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))

	if err != nil {
		log.Fatalf("Unable to get source youtube account: %v", err)
	}

	return service
}

func main() {
	ctx := context.Background()

	clientSecret, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	sourceService := getService(ctx, "source", clientSecret)
	targetService := getService(ctx, "target", clientSecret)

	handleError(err, "Error creating YouTube client")

	channelStatuses := make([]ChannelImportStatus, 100)

	// Find existing or create new channelStatuses
	if file, err := os.Open("importStatus.gob"); err == nil {
		fmt.Println("Encoded file exists, decoding into channelStatuses")
		decoder := gob.NewDecoder(file)

		channelStatuses = make([]ChannelImportStatus, 100, 250)
		decoder.Decode(&channelStatuses)

		defer file.Close()
	} else {
		fmt.Println("Encoded file doesnt exist, fetching subscriptions")
		sourceChannels, err := mySubscriptions(ctx, sourceService, []string{"snippet", "contentDetails"})

		if err != nil {
			log.Fatalf("Unable to list source channels: %v", err)
		}

		channelStatuses = make([]ChannelImportStatus, 100, len(sourceChannels))

		for index, channel := range sourceChannels {
			channelStatuses[index] = ChannelImportStatus{channel, false}
		}
	}

	// Import the unimported channels 1 by 1
	for _, channelStatus := range channelStatuses {
		channel := channelStatus.Channel

		if channelStatus.Imported {
			fmt.Printf("Channel %s already imported, skipping\n", channel.Snippet.Title)
		} else {
			fmt.Printf("Adding channel %s\n", channel.Snippet.Title)

			call := targetService.Subscriptions.Insert([]string{"snippet", "contentDetails"}, channel)
			_, err := call.Do()

			if err == nil {
				fmt.Printf("Successfully subscribed to channel: %s", channel.Snippet.Title)
				channelStatus.Imported = true
			} else {
				fmt.Printf("Unable to add channel: %v\n", err)
			}
		}

	}

	// Write status to file
	encodeFile, err := os.Create("importStatus.gob")

	if err != nil {
		panic(err)
	}

	encoder := gob.NewEncoder(encodeFile)

	if err := encoder.Encode(channelStatuses); err != nil {
		panic(err)
	}
	encodeFile.Close()
}
