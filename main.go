package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"golang.org/x/net/context"
	"os"
	"strings"
)

type source struct {
	myToken *oauth2.Token
}

func (t source) Token() (*oauth2.Token, error) {
	return t.myToken, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("missing github access token as arg 1\n")
		return
	}

	var NoContext context.Context = context.TODO()
	src := source{
		myToken: &oauth2.Token{AccessToken: os.Args[1]},
	}

	client := oauth2.NewClient(NoContext, src)
	ghClient := github.NewClient(client)
	owner := "docker-library"
	repository := "official-images"

	err := labelPullsInRepo(ghClient, owner, repository, "open", "library/")
	if err != nil {
		fmt.Printf("%v", err)
		return
	}

	return
}

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

		for _, pr := range pulls {
			commitFiles, _, err := ghClient.PullRequests.ListFiles(owner, repository, *pr.Number, nil)
			if err != nil {
				fmt.Printf("%v", err)
				continue
			}

			currentLabels := []github.Label{}
			opt := &github.ListOptions{
				PerPage: 100,
			}
			for {
				lbls, pages, err := ghClient.Issues.ListLabelsByIssue(owner, repository, *pr.Number, opt)
				if err != nil {
					fmt.Printf("%v", err)
					break
				}

				currentLabels = append(currentLabels, lbls...)

				if pages.NextPage == 0 {
					break
				}
				opt.Page = pages.NextPage
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
					fmt.Printf("%v", err)
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
