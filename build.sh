#!/bin/sh

cp -r ./docker/* .
find . -type f -exec dos2unix {} \;

docker build --no-cache -t swr.ap-southeast-1.myhuaweicloud.com/dzone/rtsp-simple-server:$1 .
docker push swr.ap-southeast-1.myhuaweicloud.com/dzone/rtsp-simple-server:$1