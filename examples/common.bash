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
