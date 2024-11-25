#!/usr/bin/env bash

g_version=$1
g_nos_cmd=${NOSCMD}
g_root=$(pwd)/.build
g_curveadm=${g_root}/dingoadm
g_curveadm_bin=${g_curveadm}/bin
rm -rf ${g_root}

mkdir -p ${g_curveadm_bin}
cp bin/dingoadm ${g_curveadm_bin}
[[ -f .CHANGELOG ]] && cp .CHANGELOG ${g_curveadm}/CHANGELOG
(cd ${g_curveadm} && ./bin/dingoadm -v && ls -ls bin/dingoadm && [[ -f CHANGELOG ]] && cat CHANGELOG)
(cd ${g_root} && tar -zcf dingoadm-${g_version}.tar.gz dingoadm)

read -p "Do you want to upload dingoeadm-${g_version}.tar.gz to NOS? " input
case $input in
    [Yy]* )
        if [ -z ${g_nos_cmd} ]; then
            echo "nos: command not found"
            exit 1
        fi
        ${g_nos_cmd} -putfile \
            ${g_root}/dingoadm-${g_version}.tar.gz \
            dingoadm \
            -key release/dingoadm-${g_version}.tar.gz \
            -replace true
        ;;
    [Nn]* )
        exit
        ;;
    * )
        echo "Please answer yes or no."
        ;;
esac
