#!/usr/bin/env bash

check_is_installed() {
    local name="$1"
    if [[ ! `which $name` ]]; then
        echo "Please install $name before runing this script. Check docs/developing.md for more information."
        exit 1
    fi
}

debug_enabled() {
    if [[ $DEBUG != 0 ]]; then
        return 0
    else
        return 1
    fi
}

service_name() {
    echo $1 | cut -d':' -f2 | tr -d [:space:]
}

service_is_running() {
    systemctl --no-legend list-units run-*.service | grep "$1" | grep running
}

service_stop_if_running() {
    local name="$1"
    echo -n "Checking if $name container is running ... "
    if [[ $(service_is_running "$name") ]]; then
        echo "yes."
        service_name=$( systemctl --no-legend list-units run-*.service | grep "$name" | grep running | cut -d' ' -f1 )
        echo -n "Shutting down $service_name ... "
        sudo systemctl stop "$service_name"
        sudo systemctl reset-failed
        echo "ok."
    else
        echo "no."
    fi
}

start_etcd_container() {
    NODE_ADDRESS=${NODE_ADDRESS:-"0.0.0.0"}
    TMP_DIR=$(mktemp -d -t gsr-example-XXXXXX)
    DATA_DIR=${DATA_DIR:-$TMP_DIR}

    echo -n "Starting etcd container for gsr tests (data-dir: $DATA_DIR)... "
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
}

get_etcd_address() {
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
}
