#!/usr/bin/env bash

# Usage: curve gateway --mdsaddr={mdsaddr} --listen-address {listenAddr} --console-address {consoleAddr} --mountpoint {mountPoint}
# Example: curve gateway --mdsaddr=172.20.7.232:16700,172.20.7.233:16700,172.20.7.234:16700 --listen-address=:29000 --console-address=:29002 --mountpoint=/home/dingofs/cli/cli1
# Created Date: 2024-10-30
# Author: Wei Dong (jackblack369)


g_curvefs="dingo"
g_curvefs_operator="gateway"
g_mdsaddr="--mdsaddr="
g_listen_address="--listen-address="
g_console_address="--console-address="
g_mountpoint="--mountpoint="
g_entrypoint="/entrypoint.sh"

function startGateway() {
    g_mdsaddr=$g_mdsaddr$1
    g_listen_address=$g_listen_address$2
    g_console_address=$g_console_address$3
    g_mountpoint=$g_mountpoint$4

    $g_curvefs $g_curvefs_operator $g_mdsaddr $g_listen_address $g_console_address $g_mountpoint
}

startGateway "$@"

ret=$?
if [ $ret -eq 0 ]; then
    echo "START GATEWAY SUCCESS"
    exit 0
else
    echo "START GATEWAY FAILED"
    exit 1
fi
