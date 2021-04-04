package main

import (
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

type Channel struct {
	id          string
	title       string
	description string
}

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

func main() {
	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/source.json
	sourceConfig, err := google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/target.json
	targetConfig, err := google.ConfigFromJSON(b, youtube.YoutubeScope)

	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	sourceClient := getClient(ctx, sourceConfig, "source")
	targetClient := getClient(ctx, targetConfig, "target")

	sourceService, err := youtube.NewService(ctx, option.WithHTTPClient(sourceClient))

	if err != nil {
		log.Fatalf("Unable to get source youtube account: %v", err)
	}

	targetService, err := youtube.NewService(ctx, option.WithHTTPClient(targetClient))

	if err != nil {
		log.Fatalf("Unable to get target youtube account: %v", err)
	}

	handleError(err, "Error creating YouTube client")

	sourceChannels, err := mySubscriptions(ctx, sourceService, []string{"snippet", "contentDetails"})

	if err != nil {
		//log.Fatalf("Unable to list source channels: %v", err)
	}

	for _, channel := range sourceChannels {
		fmt.Printf("Adding channel %s\n", channel.Snippet.Title)

		call := targetService.Subscriptions.Insert([]string{"snippet", "contentDetails"}, channel)
		_, err := call.Do()

		if err == nil {
			fmt.Printf("Successfully subscribed to channel: %s", channel.Snippet.Title)
		} else {
			fmt.Printf("Unable to add channel: %v\n", err)
		}
	}
}
