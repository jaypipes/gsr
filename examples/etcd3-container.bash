#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
FUNC_TEST_DIR=$(cd $(dirname "$0")/ && pwd)

source $FUNC_TEST_DIR/common.bash

check_is_installed rkt

if debug_enabled; then
    set -o xtrace
fi

if debug_enabled; then
    echo "================= DEBUG ==============="
    systemctl --version
    rkt version
    echo "======================================="
fi

NODE_ADDRESS=${NODE_ADDRESS:-localhost}

echo -n "Checking and installing ACI keys for etcd ... "
sudo rkt trust --prefix=coreos.com/etcd --skip-fingerprint-review >/dev/null 2>&1
echo "ok."

echo -n "Starting etcd3 rkt pod ... "
sudo systemd-run --slice=machine --description=oslively-etcd3 rkt run coreos.com/etcd:v3.0.6 -- \
    -name=oslively -advertise-client-urls=http://${NODE_ADDRESS}:2379 \
    -initial-advertise-peer-urls=http://${NODE_ADDRESS}:2380 \
    -listen-client-urls=http://0.0.0.0:2379 \
    -listen-peer-urls=http://${NODE_ADDRESS}:2380 \
    -initial-cluster=oslively=http://${NODE_ADDRESS}:2380 >/dev/null 2>&1
echo "ok."

echo -n "Determining etcd3 endpoint address ... "

sleep_time=0

GSR_TEST_ETCD_HOST=""

until [ $sleep_time -eq 8 ]; do
    sleep $(( sleep_time++ ))
    GSR_TEST_ETCD_HOST=$(sudo rkt list | grep etcd | cut -f7 | cut -d'=' -f2)
    if [[ "$GSR_TEST_ETCD_HOST" != "" ]]; then
        echo "ok."
        break
    fi
done
echo "etcd running in container at ${GSR_TEST_ETCD_HOST}:2379."
