#!/usr/bin/env bash

# Usage: create_fs USER VOLUME SIZE
# Example: create_fs curve test 10
# Created Date: 2022-01-04
# Author: chengyi01


g_dingofs_tool="dingo"
g_dingofs_tool_operator="create fs"
g_fsname="--fsname="
g_fstype="--fstype="
g_entrypoint="/entrypoint.sh"

function createfs() {
    g_fsname=$g_fsname$1
    g_fstype=$g_fstype$2

    $g_dingofs_tool $g_dingofs_tool_operator "$g_fsname" "$g_fstype"
}

createfs "$@"

ret=$?
if [ $ret -eq 0 ]; then
    $g_entrypoint "$@"
    ret=$?
    exit $ret
else
    echo "CREATEFS FAILED"
    exit 1
fi
