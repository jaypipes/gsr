#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
DATA_DIR=$(mktemp -d -t gsr-example-XXXXXX)
ROOT_DIR=$(cd $(dirname "$0")/.. && pwd)
EXAMPLES_DIR=$ROOT_DIR/examples
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

cd $ROOT_DIR

echo -n "Building Docker image for example data service ... "
docker build -t gsr-example-data:$VERSION . -f Dockerfile.example-data 2>&1 >/dev/null
if [ $? -eq 0 ]; then
    echo "ok."
else
    echo "fail."
    exit 1
fi

echo -n "Building Docker image for example web service ... "
docker build -t gsr-example-web:$VERSION . -f Dockerfile.example-web 2>&1 >/dev/null
if [ $? -eq 0 ]; then
    echo "ok."
else
    echo "fail."
    exit 1
fi

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
if [ $? -eq 0 ]; then
    echo "ok."
else
    echo "fail."
    exit 1
fi

DATA_START=$(date +%Y-%m-%dT%H:%M:%S --date="`date` + 1 sec")

sleep 2

echo ""
echo "***********************************************************************"
echo "Logs from gsr-example-data container:"
echo "***********************************************************************"
echo ""

docker container logs gsr-example-data

echo ""
echo "***********************************************************************"
echo ""
echo -n "Starting gsr-example-web container ... "
docker run -d \
    --rm \
    --name gsr-example-web \
    -e "GSR_LOG_LEVEL=3" \
    -e "GSR_ETCD_ENDPOINTS=http://$GSR_TEST_ETCD_HOST:2379" \
    -e "GSR_ETCD_CONNECT_TIMEOUT_SECONDS=3" \
    gsr-example-web:$VERSION \
    /app/main 2>&1 >/dev/null
if [ $? -eq 0 ]; then
    echo "ok."
else
    echo "fail."
    exit 1
fi

WEB_START=$(date +%Y-%m-%dT%H:%M:%S --date="`date` + 1 sec")

sleep 2

echo ""
echo "***********************************************************************"
echo "Logs from gsr-example-web container:"
echo "***********************************************************************"
echo ""

docker container logs gsr-example-web

echo ""
echo "***********************************************************************"
echo ""
echo -n "Stopping example data containers and waiting a few seconds ... "

sleep 2

docker container stop gsr-example-data 2>&1 >/dev/null

sleep 3

echo "ok."

echo ""
echo "***********************************************************************"
echo "Logs from gsr-example-web container since $WEB_START"
echo "(You should see a notification in the web container "
echo " that the data endpoint was deleted):"
echo "***********************************************************************"
echo ""


docker container logs gsr-example-web --since $WEB_START

echo ""
echo "***********************************************************************"
echo ""
echo -n "Killing example web and etcd containers ... "

docker container kill gsr-example-web gsr-example-etcd 2>&1 >/dev/null

echo "ok."
