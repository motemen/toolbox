package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const program = "gmail-permalink-from-msgid"

var (
	cacheDir  string
	tokenFile string
)

func init() {
	d, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Sprintf("os.UserCacheDir: %v", err))
	}
	cacheDir = filepath.Join(d, program)
	tokenFile = filepath.Join(cacheDir, "token.json")
}

func getClient(config *oauth2.Config) (*http.Client, error) {
	token, err := restoreToken()
	if err != nil {
		token, err = doAuthorization(config)
		if err != nil {
			return nil, err
		}

		err = storeToken(token)
		if err != nil {
			return nil, err
		}
	}

	return config.Client(context.Background(), token), nil
}

func startCodeReceiver() (<-chan string, string, func()) {
	ch := make(chan string)

	s := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/favicon.ico" {
				http.Error(w, "Not Found", 404)
				return
			}

			if code := r.FormValue("code"); code != "" {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintln(w, "Authorized.")
				ch <- code
				return
			}
		}))

	return ch, s.URL, func() { s.Close() }
}

func doAuthorization(config *oauth2.Config) (*oauth2.Token, error) {
	ch, url, cancel := startCodeReceiver()
	defer cancel()

	config.RedirectURL = url

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit below to authorize %s:\n%s\n", program, authURL)

	authCode := <-ch

	return config.Exchange(context.TODO(), authCode)
}

func restoreToken() (*oauth2.Token, error) {
	f, err := os.Open(tokenFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var token oauth2.Token
	err = json.NewDecoder(f).Decode(&token)
	return &token, err
}

func storeToken(token *oauth2.Token) error {
	err := os.MkdirAll(filepath.Dir(tokenFile), 0o777)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

func main() {
	msgid := flag.String("msgid", "", "Message-Id")
	credentialsFile := flag.String("secret", "", "path to credentials.json")
	flag.Parse()

	ctx := context.Background()

	b, err := os.ReadFile(*credentialsFile)
	if err != nil {
		log.Fatalf("-secret: %v", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Error in google.ConfigFromJSON: %v", err)
	}

	client, err := getClient(config)
	if err != nil {
		log.Fatalf("Failed to building client: %v", err)
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Failed to build Gmail service: %v", err)
	}

	q := "rfc822msgid:" + *msgid
	r, err := srv.Users.Messages.List("me").Q(q).Do()
	if err != nil {
		log.Fatalf("Failed to query %q: %v", q, err)
	}

	if len(r.Messages) == 0 {
		log.Fatalf("No message found on query %q", q)
	}

	id := r.Messages[0].Id
	fmt.Printf("https://mail.google.com/mail/#inbox/%s\n", id)
}
