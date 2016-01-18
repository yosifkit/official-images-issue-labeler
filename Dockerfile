FROM golang:1.5

WORKDIR /go/src/app
COPY *.go ./

RUN go-wrapper download
RUN go-wrapper install

ENTRYPOINT ["app"]
