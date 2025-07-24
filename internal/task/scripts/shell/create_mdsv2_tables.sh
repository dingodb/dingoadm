#!/usr/bin/env bash

# Usage: create_mdsv2_tables MdsV2BinPath MdsV2ConfPath

g_mdsv2_client=$1

# Log function with timestamp
function log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Error handling function
function error_exit() {
    log "ERROR: $1"
    exit 1
}

# Usage: initCoorList MdsV2ConfDir
#function init_coor_list() {
#    # Check if COORDINATOR_ADDR is set
#    if [ -z "$COORDINATOR_ADDR" ]; then
#      error_exit "COORDINATOR_ADDR environment variable is not set"
#    fi
#
#    log "Initializing coordinator list at $g_mdsv2_conf"
#    echo "$COORDINATOR_ADDR" > "$g_mdsv2_conf" || error_exit "Failed to write to $g_mdsv2_conf"
#    log "Coordinator list initialized successfully"
#}

function create_tables() {
    # Check if binary exists and is executable
    if [ ! -x "$g_mdsv2_client" ]; then
      error_exit "MdsV2 client binary not found or not executable: $g_mdsv2_client"
    fi

    # create tables
    echo "Creating MDSv2 tables..."
    $g_mdsv2_client --cmd=CreateAllTable --coor_addr=list://$COORDINATOR_ADDR
    local ret=$?
    if [ $ret -ne 0 ]; then
      error_exit "Failed to create MDSv2 tables (return code: $ret)"
    fi
    log "MDSv2 tables created successfully"
}

# Main execution
log "Starting MDSv2 tables creation process"
#init_coor_list
create_tables
log "All operations completed successfully"

