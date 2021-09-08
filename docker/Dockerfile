FROM golang:1.16-alpine3.13

RUN apk add --no-cache git
RUN apk add --no-cache ffmpeg

WORKDIR /s

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN export CGO_ENABLED=0 \
    && go build -o /rtsp-simple-server \
    && cp rtsp* ../ \
    && rm -rf * \
    && mv ../rtsp* ./ \
    && ls -l

ENTRYPOINT [ "/s/rtsp-simple-server" ]