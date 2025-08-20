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
g_client_binary="/dingofs/client/sbin/dingo-fuse"
g_client_config="/dingofs/client/conf/client.conf"
g_tool_config="/etc/dingo/dingo.yaml"
g_fuse_args=""
new_dingo="false"

# quota
capacity=""
inodes=""
args=()

# Parse command parameters
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
    mdsaddr=$(grep 'mdsOpt.rpcRetryOpt.addrs' "${g_client_config}" | awk -F '=' '{print $2}')
        
    # Get the metric port from the mountpoint list
    mnt_info=$(${g_dingofs_tool} list mountpoint --mdsaddr=${mdsaddr} | grep ${g_mnt} | grep $(hostname))

    # check if mnt_info is empty, skip the following steps
    if [ -z "$mnt_info" ]; then
        echo "current have not mountpoint on $(hostname), skip umount..."
    else
        echo "avoid mountpoint conflict, begin umount mountpoint on $(hostname)..."
        metric_port=$(echo "$mnt_info" | awk -F '[:]' '{print $2}')
        echo "mountpoint ${g_mnt} metric_port is ${metric_port}"
        ${g_dingofs_tool} umount fs ${g_fsname} --mountpoint $(hostname):${metric_port}:${g_mnt} --mdsaddr=${mdsaddr}
    
        # check above command is successful or not
        if [ $? -ne 0 ]; then
            echo "umount mountpoint failed, exit..."
            exit 1
        fi
    fi

    # check if mountpoint path is transport endpoint is not connected, execute umount 
    
}

function checkfs() {
    g_fsname=$g_fsname$1

    # check fs command: dingo config get --fsname <fsname>
    echo "\ncheck fs command: $g_dingofs_tool config get $g_fsname"
    $g_dingofs_tool config get "$g_fsname"
    
    if [ $? -ne 0 ]; then
        echo "FS:[$1] does not exist, now create fs:[$1]."
        createfs "$1" "$2"
        if [ $? -ne 0 ]; then
            echo "Create fs:[$1] failed, exiting..."
            return 1
        else
            echo "Create fs:[$1] successfully."
            return 0
        fi
    fi
}

function createfs() {
    g_fsname=$g_fsname$1

    if [ "$new_dingo" == "true" ]; then
        # create fs command: dingo create fs --fsname <fsname> --storagetype <storagetype> xxx
        storagetype=$(grep 'storagetype:' "${g_tool_config}" | awk '{print $2}')
        g_fstype=$g_storagetype$storagetype
    else
        # create fs command: dingo create fs --fsname <fsname> --fstype s3 xxx
        g_fstype=$g_fstype$2
    fi
    echo "\ncreate fs command: $g_dingofs_tool $g_dingofs_tool_operator $g_fsname $g_fstype"
    $g_dingofs_tool $g_dingofs_tool_operator "$g_fsname" "$g_fstype"
}

#function configfs() {
#    # config fs quota: dingo config fs --fsname <fsname>  --capacity <capacity> --inodes <inodes>
#    echo "$g_dingofs_tool $g_dingofs_tool_config $g_fsname $g_quota_capacity$1 $g_quota_inodes$2"
#    $g_dingofs_tool $g_dingofs_tool_config "$g_fsname" "$g_quota_capacity$1" "$g_quota_inodes$2" 
#}

function get_options() {
    local long_opts="role:,args:,help"
    local fuse_args=`getopt -o ra --long $long_opts -n "$0" -- "$@"`
    eval set -- "${fuse_args}"
    while true
    do
        case "$1" in
            -r|--role)
                shift 2
                ;;
            -a|--args)
                g_fuse_args=$2
                shift 2
                ;;
            -h)
                usage
                exit 1
                ;;
            --)
                shift
                break
                ;;
            *)
                exit 1
                ;;
        esac
    done
}

echo "mount fs args: ${args[@]}"
# fetch args last element as mountpoint
g_mnt=$(echo "${args[@]}" | awk '{print $NF}')
echo "mountpoint is ${g_mnt}"
if [[ ! -d "${g_mnt}" ]]; then
    echo "Mountpoint ${g_mnt} does not exist, creating it..."
    mkdir -p "${g_mnt}"
fi

cleanMountpoint
# check fs is exist or not
checkfs "${args[@]}"
ret=$?

if [ $ret -eq 0 ]; then
    if [[ -n "$capacity" && "$capacity" -ne 0 ]]; then
        echo "\nConfig fs quota: capacity=$capacity"
        $g_dingofs_tool $g_dingofs_tool_config "$g_fsname" "$g_quota_capacity$capacity"
    fi
    if [ $? -ne 0 ]; then
        echo "Config fs quota failed, exiting..."
        exit 1
    fi
    if [[ -n "$inodes" && "$inodes" -ne 0 ]]; then
        echo "Config fs quota: inode=$inodes"
        $g_dingofs_tool $g_dingofs_tool_config "$g_fsname" "$g_quota_inodes$inodes"
    fi
    if [ $? -ne 0 ]; then
        echo "Config fs quota failed, exiting..."
        exit 1
    fi
    if [ "$new_dingo" == "true" ]; then
        get_options "${args[@]}"
        echo "\nBootstrap dingo-fuse service, command: $g_client_binary ${g_fuse_args}"
        exec $g_client_binary ${g_fuse_args}
    else
        echo "\nold dingo, using entrypoint script: $g_entrypoint"
        $g_entrypoint "${args[@]}"
    fi
    ret=$?
    exit $ret
else
    echo "Check FS FAILED"
    exit 1
fi
