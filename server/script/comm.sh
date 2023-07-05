
OS_ARCH=$(uname -m)
DASHBOARD_LIST=(
"A-Ops Home Page.json" "System Inspection.json" \
"App Performance Diagnose.json" "System Performance Diagnose.json" \
"IO Full Stack - Block and Proc Metrics.json" "ThreadProfiling.json" \
"IO Full Stack - Tcp Metrics.json" "ThreadProfiling-EventDetail.json" \
"IO Full Stack.json" "Topo Graph - App Diagnose.json" \
"JVM Metrics.json" "Topo Graph - Resource.json" \
"System Flame.json" "Topo Graph.json"
)

function yum_download() {
    rpm="$@"
    repo_path=""

    if echo $REMOTE_REPO_PREFIX | grep -q "openEuler-22.03-LTS-SP1" ; then
        EPOL_REPO=$REMOTE_REPO_PREFIX/EPOL/main/${OS_ARCH}
        EPOL_UPDATE_REPO=$REMOTE_REPO_PREFIX/EPOL/update/main/${OS_ARCH}
    else
        EPOL_REPO=$REMOTE_REPO_PREFIX/EPOL/${OS_ARCH}
        EPOL_UPDATE_REPO=$REMOTE_REPO_PREFIX/EPOL/update/${OS_ARCH}
    fi

    repo_path="--repofrompath=epol_deploy,$EPOL_REPO \
        --repofrompath=epol_update_deploy,$EPOL_UPDATE_REPO \
        --repofrompath=everything_deploy,$REMOTE_REPO_PREFIX/everything/${OS_ARCH} \
        --repofrompath=update_deploy,$REMOTE_REPO_PREFIX/update/${OS_ARCH}"

    echo $repo_path
    yumdownloader --resolve $rpm $repo_path --destdir=${DOWNLOAD_DIR}  --installroot=${DOWNLOAD_DIR} --forcearch=${OS_ARCH} --nogpgcheck -b
    [ $? -ne 0 ] && echo_err_exit "Error: failed to download $rpm, please check repo!"
}


download_grafana_dashboard() {
    download_dir=$1
    for ele in "${DASHBOARD_LIST[@]}"
    do
        echo $ele 
        wget "https://gitee.com/openeuler/gala-docs/raw/master/grafana/dashboard/${ele}" \
            -O  "${download_dir}/${ele}" --no-check-certificate
        [ $? -ne 0 ] && echo_err_exit "Failed to download ${ele}"
    done
}