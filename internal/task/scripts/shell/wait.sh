#!/usr/bin/env bash

# Usage: wait ADDR...
# Example: wait 10.0.10.1:2379 10.0.10.2:2379
# Created Date: 2021-11-25
# Author: Jingli Chen (Wine93)


[[ -z $(which curl) ]] && apt-get install -y curl
wait=0
while ((wait<30))
do
    for addr in "$@"
    do
        echo "connect [$addr]..." >> /curvefs/tools/logs/wait.log
        # curl --connect-timeout 3 --max-time 10 $addr -Iso /dev/null
        curl -sfI --connect-timeout 3 --max-time 5 "$addr" > /dev/null 2>&1
        if [ $? == 0 ]; then
            echo "connect [$addr] success !" >> /curvefs/tools/logs/wait.log
            exit 0
        fi
    done
    sleep 1s
    wait=$(expr $wait + 1)
    date >> /curvefs/tools/logs/wait.log
done
echo "wait timeout"
exit 1
