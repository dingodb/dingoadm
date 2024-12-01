#!/usr/bin/env bash

# Usage: create_fs USER VOLUME SIZE
# Example: create_fs curve test 10
# Created Date: 2022-01-04
# Author: chengyi01


g_dingofs_tool="dingo"
g_dingofs_tool_operator="create fs"
g_dingofs_tool_config="config fs"
g_fsname="--fsname="
g_fstype="--fstype="
g_quota_capacity="--capacity="
g_quota_inodes="--inodes="
g_entrypoint="/entrypoint.sh"

function createfs() {
    g_fsname=$g_fsname$1
    g_fstype=$g_fstype$2

    # create fs command: dingo create fs --fsname <fsname> --fstype s3 xxx
    $g_dingofs_tool $g_dingofs_tool_operator "$g_fsname" "$g_fstype"
}

function configfs() {
    # config fs quota: dingo config fs --fsname <fsname>  --capacity <capacity> --inodes <inodes>
    echo "$g_dingofs_tool $g_dingofs_tool_config $g_fsname $g_quota_capacity$1 $g_quota_inodes$2"
    $g_dingofs_tool $g_dingofs_tool_config "$g_fsname" "$g_quota_capacity$1" "$g_quota_inodes$2" 
}

# Parse command parameters
capacity=""
inodes=""
args=()
for arg in "$@"; do
    case $arg in
        --capacity=*)
            capacity="${arg#*=}"
            ;;
        --inodes=*)
            inodes="${arg#*=}"
            ;;
        *)
            args+=("$arg")
            ;;
    esac
done

echo "create fs args: ${args[@]}"
createfs "${args[@]}"

ret=$?
if [ $ret -eq 0 ]; then
    if [ -n "$capacity" ] || [ -n "$inodes" ]; then
        echo "config fs quota: capacity=$capacity, inodes=$inodes"
        configfs "$capacity" "$inodes"
    fi
    $g_entrypoint "${args[@]}"
    ret=$?
    exit $ret
else
    echo "CREATEFS FAILED"
    exit 1
fi
