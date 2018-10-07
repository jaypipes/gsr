#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
DATA_DIR=$(mktemp -d -t gsr-example-XXXXXX)
EXAMPLES_DIR=$(cd $(dirname "$0")/ && pwd)

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

echo "Building Docker image for example data service ... "
cd $EXAMPLES_DIR/cmd/data
docker build -t gsr-example-data . -f Dockerfile

echo "Building Docker image for example web service ... "
cd $EXAMPLES_DIR/cmd/web
docker build -t gsr-example-web . -f Dockerfile

NODE_ADDRESS=${NODE_ADDRESS:-"0.0.0.0"}

echo -n "Starting etcd container for gsr tests ... "
docker run -d \
    --rm \
    -p 2379:2379 \
    -p 2380:2380 \
    --volume=${DATA_DIR}:/etcd-data \
    --name gsr-example-etcd \
    quay.io/coreos/etcd:latest \
    /usr/local/bin/etcd \
    --data-dir=/etcd-data --name node1 \
    --initial-advertise-peer-urls http://${NODE_ADDRESS}:2380 \
    --listen-peer-urls http://${NODE_ADDRESS}:2380 \
    --advertise-client-urls http://${NODE_ADDRESS}:2379 \
    --listen-client-urls http://${NODE_ADDRESS}:2379 \
    --initial-cluster node1=http://${NODE_ADDRESS}:2380 >/dev/null 2>&1
echo "ok."

echo -n "Determining etcd3 endpoint address ... "

sleep_time=0

GSR_TEST_ETCD_HOST=""

until [ $sleep_time -eq 8 ]; do
    sleep $(( sleep_time++ ))
    GSR_TEST_ETCD_HOST=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' gsr-example-etcd)
    if [[ "$GSR_TEST_ETCD_HOST" != "" ]]; then
        echo "ok."
        break
    fi
done
echo "etcd running in container at ${GSR_TEST_ETCD_HOST}:2379."

echo -n "Starting gsr-example-data container ... "
docker run -d \
    --rm \
    --name gsr-example-data \
    -e "GSR_LOG_LEVEL=3" \
    -e "GSR_ETCD_ENDPOINTS=http://$GSR_TEST_ETCD_HOST:2379" \
    -e "GSR_ETCD_CONNECT_TIMEOUT_SECONDS=3" \
    gsr-example-data:latest \
    /app/main
echo "ok."

echo -n "Starting gsr-example-web container ... "
docker run -d \
    --rm \
    --name gsr-example-web \
    -e "GSR_LOG_LEVEL=3" \
    -e "GSR_ETCD_ENDPOINTS=http://$GSR_TEST_ETCD_HOST:2379" \
    -e "GSR_ETCD_CONNECT_TIMEOUT_SECONDS=3" \
    gsr-example-web:latest \
    /app/main
echo "ok."

sleep 5

echo "Logs from gsr-example-data container:"

docker container logs gsr-example-data

echo "Logs from gsr-example-web container:"

docker container logs gsr-example-web
