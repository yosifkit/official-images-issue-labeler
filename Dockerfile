FROM golang:1.5

WORKDIR /go/src/app
COPY *.go ./

RUN go-wrapper download
RUN set -x \
	&& go-wrapper install \
	&& ! app \
	&& ! app bogus \
	&& ! app --bogus \
	&& ! app --token bogus bogus \
	&& ! app --token bogus --bogus \
	&& app --help \
	&& app -h

ENTRYPOINT ["app"]
