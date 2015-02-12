package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"os"
)

type source struct {
	myToken *oauth2.Token
}

func (t source) Token() (*oauth2.Token, error) {
	return t.myToken, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("missing github access token as avg 1\n")
		return
	}
	var NoContext oauth2.Context = nil
	src := source{
		myToken: &oauth2.Token{AccessToken: os.Args[1]},
	}

	client := oauth2.NewClient(NoContext, src)
	ghClient := github.NewClient(client)

	repos, _, err := ghClient.Repositories.List("", nil)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	for key, value := range repos {
		fmt.Printf("%d, %v\n", key, value)
	}

	return
}
