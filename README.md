# Official Images Automatic Issue Labeler [![Build Status](https://travis-ci.org/yosifkit/official-images-issue-labeler.svg?branch=master)](https://travis-ci.org/yosifkit/official-images-issue-labeler)

This is for labeling issues and PR's according to the library files they modify.

Put a Github api-key in your home directory.

```console
$ docker build -t official-images-issue-labeler .
$ docker run -it --rm official-images-issue-labeler --help
Usage:
  app [OPTIONS]

Application Options:
      --token=deadbeef             GitHub API access token
      --owner=docker-library
      --repo=official-images
      --state=[open|closed|all]

Help Options:
  -h, --help                       Show this help message

$ docker run -it --rm official-images-issue-labeler --token "$(cat ~/.github)"
```
