#!/usr/bin/env bash

DEBUG=${DEBUG:-0}
EXAMPLES_DIR=$(cd $(dirname "$0")/ && pwd)

source $EXAMPLES_DIR/common.bash

check_is_installed docker

if debug_enabled; then
    set -o xtrace
fi

start_etcd_container && get_etcd_address

echo "etcd running in container at ${GSR_TEST_ETCD_HOST}:2379."
