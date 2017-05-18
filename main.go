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

const (
	defaultFilePrefix   = "library/"
	defaultNewFileLabel = "new-image"
)

func (f labelerFlags) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: f.GhToken}, nil
}

var ghContext = context.TODO()

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

	// "The returned client is not valid beyond the lifetime of the context."
	oauthClient := oauth2.NewClient(ghContext, opts) // https://godoc.org/golang.org/x/oauth2#NewClient
	ghClient := github.NewClient(oauthClient)

	if err := labelPullsInRepo(ghClient, opts.Owner, opts.Repo, opts.State, defaultFilePrefix, defaultNewFileLabel); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

const defaultTries = 3

func listPulls(ghClient *github.Client, owner string, repository string, state string) ([]*github.PullRequest, error) {
	options := &github.PullRequestListOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	allPulls := []*github.PullRequest{}
	tries := defaultTries
	for {
		pulls, resp, err := ghClient.PullRequests.List(ghContext, owner, repository, options)
		if err != nil {
			tries--
			if tries <= 0 {
				return nil, err
			}
			continue
		}
		allPulls = append(allPulls, pulls...)
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
		tries = defaultTries
	}
	return allPulls, nil
}

func listFiles(ghClient *github.Client, owner string, repository string, pr *github.PullRequest) ([]*github.CommitFile, error) {
	options := &github.ListOptions{
		PerPage: 100,
	}
	allFiles := []*github.CommitFile{}
	tries := defaultTries
	for {
		files, resp, err := ghClient.PullRequests.ListFiles(ghContext, owner, repository, *pr.Number, options)
		if err != nil {
			tries--
			if tries <= 0 {
				return nil, err
			}
			continue
		}
		allFiles = append(allFiles, files...)
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
		tries = defaultTries
	}
	return allFiles, nil
}

func listLabels(ghClient *github.Client, owner string, repository string, pr *github.PullRequest) ([]*github.Label, error) {
	options := &github.ListOptions{
		PerPage: 100,
	}
	allLabels := []*github.Label{}
	tries := defaultTries
	for {
		files, resp, err := ghClient.Issues.ListLabelsByIssue(ghContext, owner, repository, *pr.Number, options)
		if err != nil {
			tries--
			if tries <= 0 {
				return nil, err
			}
			continue
		}
		allLabels = append(allLabels, files...)
		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
		tries = defaultTries
	}
	return allLabels, nil
}

func labelPullsInRepo(ghClient *github.Client, owner string, repository string, state string, filePrefix string, newFileLabel string) error {
	pulls, err := listPulls(ghClient, owner, repository, state)
	if err != nil {
		return err
	}

NextPull:
	for _, pr := range pulls {
		commitFiles, err := listFiles(ghClient, owner, repository, pr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error in ListFiles(%d): %v\n", *pr.Number, err)
			continue
		}

		currentLabels, err := listLabels(ghClient, owner, repository, pr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error in ListLabels(%d): %v\n", *pr.Number, err)
			continue
		}

		labels := []string{}

		for _, commitFile := range commitFiles {
			if strings.HasPrefix(*commitFile.Filename, filePrefix) {
				toAdd := []string{*commitFile.Filename}
				if commitFile.Status != nil && *commitFile.Status == "added" { // "added", "modified", etc
					toAdd = append(toAdd, newFileLabel)
				}
			LabelToAdd:
				for _, lblToAdd := range toAdd {
					for _, lbl := range currentLabels {
						if *lbl.Name == lblToAdd {
							continue LabelToAdd
						}
					}
					labels = append(labels, lblToAdd)
				}
			}
		}

		fmt.Printf("\nhttps://github.com/%s/%s/pull/%d\n\tnew:%v\n\tcurrent:%v\n", owner, repository, *pr.Number, labels, currentLabels)

		if len(labels) > 0 {
			tries := defaultTries
			for {
				labelObjs, _, err := ghClient.Issues.AddLabelsToIssue(ghContext, owner, repository, *pr.Number, labels)
				if err != nil {
					tries--
					if tries <= 0 {
						fmt.Fprintf(os.Stderr, "error in AddLabels(%d, %v): %v\n", *pr.Number, labels, err)
						continue NextPull
					}
					continue
				}
				fmt.Printf("\tresult:%v\n", labelObjs)
				break
			}
		}
	}

	fmt.Printf("\n")

	return nil
}
