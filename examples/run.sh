#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
DATA_DIR=$(mktemp -d -t gsr-example-XXXXXX)
EXAMPLES_DIR=$(cd $(dirname "$0")/ && pwd)
VERSION=`git describe --tags --always --dirty`

source $EXAMPLES_DIR/common.bash

check_is_installed docker

if debug_enabled; then
    set -o xtrace
fi

if debug_enabled; then
    echo "================= DEBUG ==============="
    systemctl --version
    docker version
    echo "======================================="
fi

echo -n "Building Docker image for example data service ... "
cd $EXAMPLES_DIR/cmd/data
docker build -t gsr-example-data:$VERSION . -f Dockerfile 2>&1 >/dev/null
echo "ok."

echo -n "Building Docker image for example web service ... "
cd $EXAMPLES_DIR/cmd/web
docker build -t gsr-example-web:$VERSION . -f Dockerfile 2>&1 >/dev/null
echo "ok."

start_etcd_container && get_etcd_address

echo "etcd running in container at ${GSR_TEST_ETCD_HOST}:2379."

echo -n "Starting gsr-example-data container ... "
docker run -d \
    --rm \
    --name gsr-example-data \
    -e "GSR_LOG_LEVEL=3" \
    -e "GSR_ETCD_ENDPOINTS=http://$GSR_TEST_ETCD_HOST:2379" \
    -e "GSR_ETCD_CONNECT_TIMEOUT_SECONDS=3" \
    gsr-example-data:$VERSION \
    /app/main 2>&1 >/dev/null
echo "ok."

DATA_START=$(date +%Y-%m-%dT%H:%M:%S --date="`date` + 1 sec")

echo "Logs from gsr-example-data container:"

docker container logs gsr-example-data

echo -n "Starting gsr-example-web container ... "
docker run -d \
    --rm \
    --name gsr-example-web \
    -e "GSR_LOG_LEVEL=3" \
    -e "GSR_ETCD_ENDPOINTS=http://$GSR_TEST_ETCD_HOST:2379" \
    -e "GSR_ETCD_CONNECT_TIMEOUT_SECONDS=3" \
    gsr-example-web:$VERSION \
    /app/main 2>&1 >/dev/null
echo "ok."

WEB_START=$(date +%Y-%m-%dT%H:%M:%S --date="`date` + 1 sec")

echo "Logs from gsr-example-web container:"

docker container logs gsr-example-web

echo "Logs from gsr-example-data container since $DATA_START:"

docker container logs gsr-example-data --since $DATA_START

echo -n "Killing example data containers and waiting 10 seconds ... "

docker container kill gsr-example-data 2>&1 >/dev/null

sleep 10

echo "ok."

echo "Logs from gsr-example-web container since $WEB_START:"

docker container logs gsr-example-web --since $WEB_START

echo -n "Killing example web and etcd containers ... "

docker container kill gsr-example-web gsr-example-etcd 2>&1 >/dev/null

echo "ok."
