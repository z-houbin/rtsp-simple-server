#!/bin/sh

docker build --no-cache -t swr.ap-southeast-1.myhuaweicloud.com/dzone/rtsp-simple-server:$1 .
docker push swr.ap-southeast-1.myhuaweicloud.com/dzone/rtsp-simple-server:$1