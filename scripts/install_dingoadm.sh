#!/usr/bin/env bash

############################  GLOBAL VARIABLES
g_color_yellow=$(printf '\033[33m')
g_color_red=$(printf '\033[31m')
g_color_normal=$(printf '\033[0m')
g_dingoadm_home="${HOME}/.dingoadm"
g_bin_dir="${g_dingoadm_home}/bin"
g_db_path="${g_dingoadm_home}/data/dingoadm.db"
g_profile="${HOME}/.profile"
g_root_url="https://work.dingodb.top"
g_upgrade="${CURVEADM_UPGRADE}"
g_version="${CURVEADM_VERSION:=$g_latest_version}"
g_download_url="${g_root_url}/dingoadm.tar.gz"

############################  BASIC FUNCTIONS
msg() {
    printf '%b' "${1}" >&2
}

success() {
    msg "${g_color_yellow}[✔]${g_color_normal} ${1}${2}"
}

die() {
    msg "${g_color_red}[✘]${g_color_normal} ${1}${2}"
    exit 1
}

program_must_exist() {
    local ret='0'
    command -v "${1}" >/dev/null 2>&1 || { local ret='1'; }

    if [ "${ret}" -ne 0 ]; then
        die "You must have '$1' installed to continue.\n"
    fi
}

############################ FUNCTIONS
backup() {
    if [ -d "${g_dingoadm_home}" ]; then
        mv "${g_dingoadm_home}" "${g_dingoadm_home}-$(date +%s).backup"
    fi
}

setup() {
    mkdir -p "${g_dingoadm_home}"/{bin,data,module,logs,temp}

    # generate config file
    local confpath="${g_dingoadm_home}/dingoadm.cfg"
    if [ ! -f "${confpath}" ]; then
        cat << __EOF__ > "${confpath}"
[defaults]
log_level = error
sudo_alias = "sudo"
timeout = 300
auto_upgrade = false

[ssh_connections]
retries = 3
timeout = 10

[database]
url = "${g_db_path}"
__EOF__
    fi
}

install_binray() {
    local ret=1
    local tempfile="/tmp/dingoadm-$(date +%s%6N).tar.gz"
    curl "${g_download_url}" -skLo "${tempfile}"
    if [ $? -eq 0 ]; then
        tar -zxvf "${tempfile}" -C "${g_bin_dir}" 1>/dev/null
        ret=$?
    fi

    # rm  "${tempfile}"
    if [ ${ret} -eq 0 ]; then
        chmod 755 "${g_bin_dir}/dingoadm"
    else
        die "Download dingoadm failed\n"
    fi
}

set_profile() {
    shell=$(echo "$SHELL" | awk 'BEGIN {FS="/";} { print $NF }')
    if [ -f "${HOME}/.${shell}_profile" ]; then
        g_profile="${HOME}/.${shell}_profile"
    elif [ -f "${HOME}/.${shell}_login" ]; then
        g_profile="${HOME}/.${shell}_login"
    elif [ -f "${HOME}/.${shell}rc" ]; then
        g_profile="${HOME}/.${shell}rc"
    fi

    case :${PATH}: in
        *:${g_bin_dir}:*) ;;
        *) echo "export PATH=${g_bin_dir}:\${PATH}" >> "${g_profile}" ;;
    esac
}

print_install_success() {
    success "Install dingoadm ${g_version} success, please run 'source ${g_profile}'\n"
}

print_upgrade_success() {
    if [ -f "${g_dingoadm_home}/CHANGELOG" ]; then
        cat "${g_dingoadm_home}/CHANGELOG"
    fi
    success "Upgrade dingoadm to ${g_version} success\n"
}

install() {
    backup
    setup
    install_binray
    set_profile
    print_install_success
}

upgrade() {
    install_binray
    print_upgrade_success
}

main() {
    if [ "${g_upgrade}" == "true" ]; then
        upgrade
    else
        install
    fi
}

############################  MAIN()
main "$@"
