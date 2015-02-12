# Official Images Automatic Issue Labeler

This is for labeling issues and PR's according to the library files they modify.

Put a Github api-key in your home directory.

```console
$ docker build -t go-label .
$ docker run -it --rm go-label go-wrapper run "$(cat ~/.github)"
```
