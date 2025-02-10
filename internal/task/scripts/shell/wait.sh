#!/usr/bin/env bash

# Usage: wait ADDR...
# Example: wait 10.0.10.1:2379 10.0.10.2:2379
# Created Date: 2021-11-25
# Author: Jingli Chen (Wine93)


[[ -z $(which curl) ]] && apt-get install -y curl
wait=0
start_time=$(date +%s)

while ((wait<30))
do
    for addr in "$@"
    do
        echo "connect [$addr]..." >> /dingofs/tools/logs/wait.log
        # curl --connect-timeout 3 --max-time 10 $addr -Iso /dev/null
        curl -sfI --connect-timeout 3 --max-time 5 "$addr" > /dev/null 2>&1
        if [ $? == 0 ]; then
            echo "connect [$addr] success !" >> /dingofs/tools/logs/wait.log
            exit 0
        fi
        echo "connect [$addr] failed !" >> /dingofs/tools/logs/wait.log
    done

    #sleep 1s # should not sleep in jenkins pipeline embedded shell
    #wait=$(expr $wait + 1)
    
    # Replace sleep with a busy wait for 1 second
    target_time=$((start_time + wait + 1))
    while [[ $(date +%s) -lt $target_time ]]; do
        :  # Do nothing, just wait
    done

    wait=$((wait + 1))
    echo "wait 1s" >> /dingofs/tools/logs/wait.log
    date >> /dingofs/tools/logs/wait.log
done
echo "wait timeout"
exit 1
