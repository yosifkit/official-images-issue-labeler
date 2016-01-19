package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/jessevdk/go-flags"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type labelerFlags struct {
	GhToken string `long:"token" required:"true" description:"GitHub API access token" value-name:"deadbeef"`
	Owner   string `long:"owner" default:"docker-library" value-name:"docker-library"`
	Repo    string `long:"repo" default:"official-images" value-name:"official-images"`
	State   string `long:"state" default:"open" choice:"open" choice:"closed" choice:"all"`
}

func (f labelerFlags) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: f.GhToken}, nil
}

func main() {
	opts := labelerFlags{}
	flagParser := flags.NewParser(&opts, flags.Default)

	args, err := flagParser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				return
			}
		}
		os.Exit(1)
	}
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "error: unexpected arguments: %s\n\n", strings.Join(args, " "))
		flagParser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	oauthContext := context.TODO()                      // "The returned client is not valid beyond the lifetime of the context."
	oauthClient := oauth2.NewClient(oauthContext, opts) // https://godoc.org/golang.org/x/oauth2#NewClient
	ghClient := github.NewClient(oauthClient)

	if err := labelPullsInRepo(ghClient, opts.Owner, opts.Repo, opts.State, "library/"); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

const defaultTries = 3

func labelPullsInRepo(ghClient *github.Client, owner string, repository string, state string, filePrefix string) error {
	options := &github.PullRequestListOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		pulls, resp, err := ghClient.PullRequests.List(owner, repository, options)
		if err != nil {
			return err
		}

	NextPull:
		for _, pr := range pulls {
			commitFiles, _, err := ghClient.PullRequests.ListFiles(owner, repository, *pr.Number, nil)
			if err != nil {
				fmt.Printf("pr.ListFiles(%d) error: %v\n", *pr.Number, err)
				continue
			}

			currentLabels := []github.Label{}
			opt := &github.ListOptions{
				PerPage: 100,
			}
			tries := defaultTries
			for {
				lbls, pages, err := ghClient.Issues.ListLabelsByIssue(owner, repository, *pr.Number, opt)
				if err != nil {
					fmt.Printf("pr.ListLabels(%d) error: %v\n", *pr.Number, err)
					tries--
					if tries <= 0 {
						continue NextPull
					}
					continue
				}

				currentLabels = append(currentLabels, lbls...)

				if pages.NextPage == 0 {
					break
				}

				opt.Page = pages.NextPage
				tries = defaultTries
			}

			labels := []string{}
			for _, commitFile := range commitFiles {
				if strings.HasPrefix(*commitFile.Filename, filePrefix) {
					valid := true
					toAdd := *commitFile.Filename
					for _, lbl := range currentLabels {
						if lbl.String() == toAdd {
							valid = false
							break
						}
					}
					if valid {
						labels = append(labels, toAdd)
					}
				}
			}
			fmt.Printf("%d:\n\tnew:%v\n\tcurrent:%v\n", *pr.Number, labels, currentLabels)

			// add labels
			if len(labels) > 0 {
				labelObjs, _, err := ghClient.Issues.AddLabelsToIssue(owner, repository, *pr.Number, labels)
				if err != nil {
					fmt.Printf("pr.AddLabels(%d, %v) error: %v\n", *pr.Number, labels, err)
					continue
				}
				fmt.Printf("\tresult:%v\n", labelObjs)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		options.ListOptions.Page = resp.NextPage
	}

	return nil
}
