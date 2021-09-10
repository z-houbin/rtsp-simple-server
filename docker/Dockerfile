FROM golang:1.16-alpine3.13

RUN apk add --no-cache git
RUN apk add --no-cache ffmpeg

WORKDIR /s

COPY . ./

RUN chmod +x rtsp-simple-server && rm -rf *.sh

ENTRYPOINT [ "/s/rtsp-simple-server" ]