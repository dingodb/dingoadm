#!/usr/bin/env bash

# Usage: create_fs USER VOLUME SIZE

g_dingofs_tool="dingo"
g_dingofs_tool_operator="create fs"
g_dingofs_tool_config="config fs"
g_fsname="--fsname="
g_fstype="--fstype="
g_storagetype="--storagetype="
g_quota_capacity="--capacity="
g_quota_inodes="--inodes="
g_entrypoint="/entrypoint.sh"
g_mnt=""
g_tool_config="/dingofs/client/conf/client.conf"
new_dingo="false"

function cleanMountpoint(){

    # Check if mountpoint path is broken (Transport endpoint is not connected)
    if mountpoint -q "${g_mnt}"; then
        echo "Mountpoint ${g_mnt} is mounted properly. begin umount it "
        umount -l "${g_mnt}"
    elif grep -q 'Transport endpoint is not connected' < <(ls "${g_mnt}" 2>&1); then
        echo "Mountpoint ${g_mnt} is in 'Transport endpoint is not connected' state. Forcing umount..."
        fusermount -u "${g_mnt}" || umount -l "${g_mnt}"
    fi

    # Get the MDS address from the client.conf file
    mdsaddr=$(grep 'mdsOpt.rpcRetryOpt.addrs' "${g_tool_config}" | awk -F '=' '{print $2}')
        
    # Get the metric port from the mountpoint list
    mnt_info=$(${g_dingofs_tool} list mountpoint --mdsaddr=${mdsaddr} | grep ${g_mnt} | grep $(hostname))

    # check if mnt_info is empty, skip the following steps
    if [ -z "$mnt_info" ]; then
        echo "current have not mountpoint on $(hostname), skip umount..."
    else
        echo "avoid mountpoint conflict, begin umount mountpoint on $(hostname)..."
        metric_port=$(echo "$mnt_info" | awk -F '[:]' '{print $2}')
        echo "mountpoint ${g_mnt} metric_port is ${metric_port}"
        ${g_dingofs_tool} umount fs --fsname ${g_fsname} --mountpoint $(hostname):${metric_port}:${g_mnt} --mdsaddr=${mdsaddr}
    
        # check above command is successful or not
        if [ $? -ne 0 ]; then
            echo "umount mountpoint failed, exit..."
            exit 1
        fi
    fi

    # check if mountpoint path is transport endpoint is not connected, execute umount 
    
}

function createfs() {
    g_fsname=$g_fsname$1

    if [ "$new_dingo" == "true" ]; then
        # create fs command: dingo create fs --fsname <fsname> --storagetype <storagetype> xxx
        g_fstype=$g_storagetype$2
    else
        # create fs command: dingo create fs --fsname <fsname> --fstype s3 xxx
        g_fstype=$g_fstype$2
    fi
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
        --new-dingo)
            new_dingo="true"
            ;;
        *)
            args+=("$arg")
            ;;
    esac
done

echo "create fs args: ${args[@]}"
g_mnt=${!#}
cleanMountpoint
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
