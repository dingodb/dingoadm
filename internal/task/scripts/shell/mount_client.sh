
#!/usr/bin/env bash

# Usage: create_fs USER VOLUME SIZE

g_dingofs_tool="dingo"
g_fsname=""
g_entrypoint="/entrypoint.sh"
g_mdsaddr=""
g_mnt=""
g_client_binary="/dingofs/client/sbin/dingo-client"
g_client_config="/dingofs/client/conf/client.conf"
g_tool_config="/etc/dingo/dingo.yaml"
g_fuse_args=""

# quota
capacity=""
inodes=""
args=()

# Parse command parameters
for arg in "$@"; do
    case $arg in
        --fsname=*)
            g_fsname="${arg#*=}"
            ;;
        --mdsaddr=*)
            g_mdsaddr="${arg#*=}"
            ;;
        --mountpoint=*)
            g_mnt="${arg#*=}"
            ;;
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

echo "mount fs args: ${args[@]}, fsname=${g_fsname}, mdsaddr=${g_mdsaddr}, mountpoint=${g_mnt}, capacity=${capacity}, inodes=${inodes}"
if [[ ! -d "${g_mnt}" ]]; then
    echo "Mountpoint ${g_mnt} does not exist, creating it..."
    mkdir -p "${g_mnt}"
fi

function cleanMountpoint(){

    # Check if mountpoint path is broken (Transport endpoint is not connected)
    mountpoint -q "${g_mnt}"
    # check if mountpoint is mount point which code is 0
    # 0: path is a mountpoint
    # 1: path is not a mountpoint but exists
    # 32: path does not exist
    if [ $? -eq 0 ]; then
        echo "mountpoint -q ${g_mnt} return 0"
        #echo "Mountpoint ${g_mnt} is have mounted. begin umount it "
        #umount -l "${g_mnt}"
    fi
    
    if grep -q 'Transport endpoint is not connected' < <(ls "${g_mnt}" 2>&1); then
        echo "Mountpoint ${g_mnt} is in 'Transport endpoint is not connected' state. Forcing umount..."
        fusermount -u "${g_mnt}" || umount -l "${g_mnt}"
    fi

    # Get the metric port from the mountpoint list
    mnt_info=$(${g_dingofs_tool} list mountpoint --mdsaddr=${g_mdsaddr} | grep ${g_mnt} | grep $(hostname))

    # check if mnt_info is empty, skip the following steps
    if [ -z "$mnt_info" ]; then
        echo "current have not mountpoint on $(hostname), skip umount..."
    else
        echo "avoid mountpoint conflict, begin umount mountpoint on $(hostname)..."
        metric_port=$(echo "$mnt_info" | awk -F '[:]' '{print $2}')
        echo "mountpoint ${g_mnt} metric_port is ${metric_port}"
        ${g_dingofs_tool} umount fs --fsname ${g_fsname} --mountpoint $(hostname):${metric_port}:${g_mnt} --mdsaddr ${g_mdsaddr}
    
        # check above command is successful or not
        if [ $? -ne 0 ]; then
            echo "umount mountpoint failed, exit..."
            exit 1
        fi
    fi

    # check if mountpoint path is transport endpoint is not connected, execute umount 
    
}

function checkfs() {

    # check fs command: dingo config get --fsname <fsname>
    echo -e "\ncheck fs command: $g_dingofs_tool config get $g_fsname"
    $g_dingofs_tool config get --fsname "$g_fsname"

    if [ $? -ne 0 ]; then
        echo "FS:[$1] does not exist, now create fs:[$1]."
        createfs
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

    # create fs command: dingo create fs --fsname <fsname> --storagetype <storagetype> xxx
    storagetype=$(grep 'storagetype:' "${g_tool_config}" | awk '{print $2}')

    echo -e "\ncreate fs command: $g_dingofs_tool create fs --fsname $g_fsname --storagetype $storagetype"
    $g_dingofs_tool create fs --fsname "$g_fsname" 
}

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

cleanMountpoint
# check fs is exist or not
checkfs "${args[@]}"
ret=$?

if [ $ret -eq 0 ]; then
    if [[ -n "$capacity" && "$capacity" -ne 0 ]]; then
        echo -e "\nConfig fs quota: capacity=$capacity"
        $g_dingofs_tool config fs --fsname $g_fsname --capacity $capacity
    fi
    if [ $? -ne 0 ]; then
        echo "Config fs quota failed, exiting..."
        exit 1
    fi
    if [[ -n "$inodes" && "$inodes" -ne 0 ]]; then
        echo "Config fs quota: inode=$inodes"
        $g_dingofs_tool config fs --fsname $g_fsname --capacity $inodes
    fi
    if [ $? -ne 0 ]; then
        echo "Config fs quota failed, exiting..."
        exit 1
    fi

    echo -e "\nBootstrap dingo-fuse service, command: $g_client_binary --flagfile ${g_client_config} mds://${g_mdsaddr}/${g_fsname} ${g_mnt}"
    exec $g_client_binary --flagfile ${g_client_config} mds://${g_mdsaddr}/${g_fsname} ${g_mnt}

    ret=$?
    exit $ret
else
    echo "Check FS FAILED"
    exit 1
fi
