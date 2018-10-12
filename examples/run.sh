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
    gsr-example-data:latest \
    /app/main 2>&1 >/dev/null
echo "ok."

sleep 5

echo "Logs from gsr-example-data container:"

docker container logs gsr-example-data

echo -n "Starting gsr-example-web container ... "
docker run -d \
    --rm \
    --name gsr-example-web \
    -e "GSR_LOG_LEVEL=3" \
    -e "GSR_ETCD_ENDPOINTS=http://$GSR_TEST_ETCD_HOST:2379" \
    -e "GSR_ETCD_CONNECT_TIMEOUT_SECONDS=3" \
    gsr-example-web:latest \
    /app/main 2>&1 >/dev/null
echo "ok."

echo "Logs from gsr-example-web container:"

docker container logs gsr-example-web

echo "Logs from gsr-example-data container:"

docker container logs gsr-example-data

echo "Killing example data containers ... "

docker container kill gsr-example-data

echo "Logs from gsr-example-web container:"

docker container logs gsr-example-web

echo -n "Killing example web and etcd containers ... "

docker container kill gsr-example-web gsr-example-etcd 2>&1 >/dev/null

echo "ok."
