#!/bin/bash


# *******************************************************************************************************comm.sh
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
# *******************************************************************************************************
OS_TYPE=""
OS_VERSION=""
DEPLOY_TYPE="remote"
OFFICIAL_RELEASE="yes"
WORKING_DIR=$(realpath $(dirname $0))
GALA_DEPLOY_MODE="rpm"
COMPONENT=""

DOCKER_HUB='hub.oepkgs.net'
DOCKER_HUB_TAG_PREFIX="${DOCKER_HUB}/a-ops"

LOCAL_DEPLOY_SRCDIR="${WORKING_DIR}"
gopher_local_rpm=""
GOPHER_DOCKER_TAG=""
REMOTE_REPO_PREFIX="http://mirrors.aliyun.com/openeuler/"
EPOL_REPO=""
EPOL_UPDATE_REPO=""

GS_DATADIR=""

KAFKA_PORT=9092
PROMETHEUS_PORT=9090
ES_PORT=9200
ARANGODB_PORT=8529
PYROSCOPE_PORT=4040
NGINX_PORT=9995

KAFKA_ADDR="localhost:${KAFKA_PORT}"
PROMETHEUS_ADDR="localhost:${PROMETHEUS_PORT}"
ES_ADDR="localhost:${ES_PORT}"
ARANGODB_ADDR="localhost:${ARANGODB_PORT}"
PYROSCOPE_ADDR="localhost:${PYROSCOPE_PORT}"
NGINX_ADDR="localhost:${NGINX_PORT}"

PROMETHEUS_SCRAPE_LIST=""

#=======Common Utils========#
function echo_err_exit() {
    echo -e "\e[31m $@ \e[0m"
    exit 1;
}

function echo_info() {
    echo -e "\e[32m $@ \e[0m"
}

function echo_warn() {
    echo -e "\e[33m $@ \e[0m"
}

function print_usage() {
    echo "usage : sh deploy.sh [COMPONENT] [OPTION]"

    echo "supported COMPONENT:"
    echo "      gopher|ops|middleware|opengauss|grafana"
    echo ""
    echo "gopher options:"
    echo "      [-K|--kafka <kafka_server>] [-p|--pyroscope <pyroscope_server>] [--docker]"
    echo "      [--proxy]"
    echo ""
    echo "ops options:"
    echo "      [-K|--kafka <kafka_server>] [-P|--prometheus <prometheus_server>] [-A|--arangodb <arangodb_server>]"
    echo "      [--docker]"
    echo ""
    echo "opengauss options:"
    echo "      [-D|--datadir <opengauss data dir>]"
    echo ""
    echo "middleware options:"
    echo "      [-K|--kafka <kafka_server>] [-P <prometheus_addr1[,prometheus_addr2,prometheus_addr3,...]>]"
    echo "      [-E|--elastic <es_server>] [-A|--arangodb] [-p|--pyroscope]"
    echo ""
    echo "grafana options:"
    echo "      [-P|--prometheus <prometheus_server>] [-p|--pyroscope <pyroscope_server>]"
    echo "      [-E|--elastic <es_server>]"
    echo ""
    echo "Common options:"
    echo "      --docker         Deploy components with docker images, only support gopher"
    echo "      -S|--srcdir      To specify offline resources for installation, only Used in offline deployment"
    echo ""
}

function get_port_from_addr() {
    addr=$1
    port=""

    if echo $addr | grep -q ":" ; then
        port=${addr##*:}
        if [ -z "${port}" ] || ! echo $port | grep -q '^[[:digit:]]*$' ; then
            echo_err_exit "Invalid port specified: $addr"
        fi
    fi
    echo $port
}

function get_ip_from_addr() {
    addr=$1
    echo ${addr%:*}
}

function addr_add_port() {
    addr="${1}"
    default_port="${2}"

    if [ -z "${addr}" ] || [ -z "${default_port}" ] ; then
        echo_err_exit "Invalid parameter in addr_add_port()"
    fi

    port=$(get_port_from_addr $addr)
    if [ -z "${port}" ] ; then
        echo "${addr}:${default_port}"
    else
        echo "${addr}"
    fi
}

function gala_wget() {
    url=$1
    dst_dir=$2
    wget_file=""

    if [ "x$dst_dir" == "x" ] ; then
        dst_dir="./"
    fi
    wget_file=$(echo ${url##*/})
    if [ -f $dst_dir/$wget_file ] ; then
        return;
    fi

    wget $url -P $dst_dir --no-check-certificate
    [ $? -ne 0 ] && echo_err_exit "Error: fail to download $wget_file"
}

function install_rpm_local_repo() {
    rpm="$1"

    [ -z "$LOCAL_DEPLOY_SRCDIR" ] && echo_err_exit "local repo is undefined, aborting!"

    yum install -y $rpm --repofrompath="local_deploy,$LOCAL_DEPLOY_SRCDIR" --nogpgcheck
    [ $? -ne 0 ] && echo_err_exit "Error: failed to install $rpm, please check repo!"
}


function install_rpm_remote_repo() {
    rpm="$1"
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

    yum install -y $rpm $repo_path --nogpgcheck
    [ $? -ne 0 ] && echo_err_exit "Error: failed to install $rpm, please check repo!"
}

function install_rpm() {
    rpm=$1
    if echo $rpm | grep -q ".rpm$" ; then
        rpm_name="$(rpm -qpi $rpm | grep Name | awk -F : '{gsub(/[[:blank:]]*/,"",$2);print $2}')"
    else
        rpm_name="$rpm"
    fi

    if [ -n "$rpm_name" ] && rpm -q "$rpm_name" >/dev/null 2>&1 ; then
        echo_info "$rpm_name is already installed, skip..."
        return
    fi

    if [ "$DEPLOY_TYPE" == "local" ] ; then
        install_rpm_local_repo $@
    elif [ "$DEPLOY_TYPE" == "remote" ] ; then
        install_rpm_remote_repo $rpm_name
    else
        echo_err_exit "Unsupported repo type, please check!"
    fi
}

function config_docker() {
    if  ! grep "^INSECURE_REGISTRY" /etc/sysconfig/docker | grep -q "${DOCKER_HUB}" ; then
        cat >> /etc/sysconfig/docker << EOF
INSECURE_REGISTRY='--insecure-registry ${DOCKER_HUB}'
EOF
        systemctl daemon-reload
        systemctl restart docker || echo_err_exit "Error: fail to configure docker"
    fi
}


function docker_load_image_file() {
    image_tarfile="$1"

    docker --version >/dev/null 2>&1  || echo_err_exit "Error: Docker cmd not found, please install docker firstly"
    [ ! -f $image_tarfile ] && echo_err_exit "Error: failed to find local image file:" $image_tarfile
    docker load -i  $image_tarfile
    [ $? -ne 0 ] && echo_err_exit "Error: failed to load docker image:" $image_tarfile
}

function docker_pull_image() {
    tag_name="$1"

    docker --version >/dev/null 2>&1  || echo_err_exit "Error: Docker cmd not found, please install docker firstly"
    config_docker
    docker pull ${DOCKER_HUB_TAG_PREFIX}/"${tag_name}"
    [ $? -ne 0 ] && echo_err_exit "Error: failed to pull docker image:" $tag_name
}


#=======openGauss Server Deployment=======#
OPNEGAUSS_DEPLOY_SCRIPT='./opengauss/create_master_slave.sh'
function parse_arg_opengauss_server() {
    ARGS=`getopt -a -o D: --long datadir: -- "$@"`
    [ $? -ne 0 ] && (print_usage; exit 1)
    eval set -- "${ARGS}"
    while true
    do
        case $1 in
            -D|--datadir)
                GS_DATADIR="${2}"
                shift;;
            --)
                shift
                break;;
            *)
                print_usage
                exit 1;;
        esac
        shift
    done
}

function create_opengauss_master_slave() {
    GS_PASSWORD=Aops@123
    OG_SUBNET="172.11.0.0/24"
    IPPREFIX=172.11.0.
    START=101
    HOST_PORT=5432
    LOCAL_PORT=5434

    MASTER_NODENAME=opengauss_master
    SLAVE_NODENAME=opengauss_slave
    nums_of_slave=1

    docker stop ${MASTER_NODENAME} 2>/dev/null ||:
    docker rm ${MASTER_NODENAME} 2>/dev/null ||:
    for ((i=1;i<=nums_of_slave;i++)) ; do
        docker stop ${SLAVE_NODENAME}${i} 2>/dev/null ||:
        docker rm ${SLAVE_NODENAME}${i} 2>/dev/null ||:
    done
    docker network rm opengaussnetwork 2>/dev/null

    docker network create --subnet=$OG_SUBNET opengaussnetwork \
    || {
      echo ""
      echo "ERROR: OpenGauss Database Network was NOT successfully created."
      echo "HINT: opengaussnetwork Maybe Already Exsist Please Execute 'docker network rm opengaussnetwork' "
      exit 1
    }
    echo "OpenGauss Database Network Created."

    conninfo=""
    for ((i=1;i<=nums_of_slave;i++))
    do
        ip=`expr $START + $i`
        hport=`expr $HOST_PORT + 1000 \* $i`
        lport=`expr $LOCAL_PORT + 1000 \* $i`
        conninfo+="replconninfo$i = 'localhost=$IPPREFIX$START localport=$LOCAL_PORT localservice=$HOST_PORT remotehost=$IPPREFIX$ip remoteport=$lport remoteservice=$hport'\n"
    done
    echo -e $conninfo

    for ((i=0;i<=nums_of_slave;i++))
    do
        if [ $i == 0 ]; then
            hport=$HOST_PORT
            lport=$LOCOL_PORT
            ip=$START
            nodeName=$MASTER_NODENAME
            conn=$conninfo
            role="primary"
        else
            hport=`expr $HOST_PORT + 1000 \* $i`
            lport=`expr $LOCAL_PORT + 1000 \* $i`
            ip=`expr $START + $i`
            nodeName=$SLAVE_NODENAME$i
            conn="replconninfo1 = 'localhost=$IPPREFIX$ip localport=$lport localservice=$hport remotehost=$IPPREFIX$START remoteport=$LOCAL_PORT remoteservice=$HOST_PORT'\n"
            role="standby"
        fi
        docker run --network opengaussnetwork --ip $IPPREFIX$ip --privileged=true \
        --name $nodeName -h $nodeName -p $hport:$hport -d \
        -e GS_PORT=$hport \
        -e OG_SUBNET=$OG_SUBNET \
        -e GS_PASSWORD=$GS_PASSWORD \
        -e NODE_NAME=$nodeName \
        -e REPL_CONN_INFO="$conn" \
        -v $GS_DATADIR/$nodeName:/var/lib/opengauss \
        hub.oepkgs.net/a-ops/opengauss:3.0.0 -M $role \
        || echo_err_exit "ERROR: OpenGauss Database $role  Docker Container was NOT successfully created."

        echo_info "OpenGauss Database $role Docker Container created."
        sleep 30
    done
}

function deploy_opengauss_server() {
    echo_info "======Deploying openGauss Server======"
    [ "$DEPLOY_TYPE" == "local" ] && echo_err_exit "openGauss server now not support offline deployment, aborting"

    if [ -n "${GS_DATADIR}" ] && [ ! -d "${GS_DATADIR}" ] ; then
        echo_err_exit "Invalid openGauss data dir"
    fi

    echo -e "[1] Pulling opengauss docker image"
    docker_pull_image "opengauss:3.0.0"

    echo -e "\n[2] Creating opengauss master and slave container"
    create_opengauss_master_slave

    echo -e "\n[4] Creating opengauss database and user"
    systemctl restart docker   # prevent iptables-related issues
    container_id=$(docker ps | grep -w opengauss_master | awk '{print $1}')
    docker exec -it ${container_id} /bin/bash -c  "su - omm -c \"echo create database tpccdb\; > ~/tmp.gsql\""
    docker exec -it ${container_id} /bin/bash -c  "su - omm -c \"echo create user tpcc with password \'tpcc_123456\'\; >> ~/tmp.gsql\""
    docker exec -it ${container_id} /bin/bash -c  "su - omm -c \"echo grant all privilege to tpcc\; >> ~/tmp.gsql\""
    docker exec -it ${container_id} /bin/bash -c  "su - omm -c \"echo create user opengauss_exporter with monadmin password \'opengauss_exporter123\'\; >> ~/tmp.gsql\""
    docker exec -it ${container_id} /bin/bash -c  "su - omm -c \"echo grant usage on schema dbe_perf to opengauss_exporter\; >> ~/tmp.gsql\""
    docker exec -it ${container_id} /bin/bash -c  "su - omm -c \"echo grant select on pg_stat_replication to opengauss_exporter\; >> ~/tmp.gsql\""

    docker exec -it ${container_id} /bin/bash -c  "su - omm -c \"gsql -f ~/tmp.gsql >/dev/null\""
    echo_info "======Deploying openGauss Server Done!======"
}

#=======Gopher Deployment=======#
GOPHER_CONF='/etc/gala-gopher/gala-gopher.conf'
GOPHER_APP_CONF='/etc/gala-gopher/gala-gopher-app.conf'
PG_STAT_CONF='/etc/gala-gopher/extend_probes/pg_stat_probe.conf'
STACKPROBE_CONF='/etc/gala-gopher/extend_probes/stackprobe.conf'
function parse_arg_gopher() {
    ARGS=`getopt -a -o K:p:S: --long kafka:,pyroscope:,docker,srcdir: -- "$@"`
    [ $? -ne 0 ] && (print_usage; exit 1)
    eval set -- "${ARGS}"
    while true
    do
        case $1 in
            -K|--kafka)
                KAFKA_ADDR=$(addr_add_port $2 ${KAFKA_PORT})
                shift;;
            -p|--pyroscope)
                PYROSCOPE_ADDR=$(addr_add_port $2 ${PYROSCOPE_PORT})
                shift;;
            -S|--srcdir)
                DEPLOY_TYPE="local"
                LOCAL_DEPLOY_SRCDIR=$(realpath $2)
                shift;;
            --docker)
                GALA_DEPLOY_MODE="docker"
                ;;
            --)
                shift
                break;;
            *)
                print_usage
                exit 1;;
        esac
        shift
    done
}

download_gopher_deps() {
    DOWNLOAD_DIR=$1

    gala_wget https://mirrors.aliyun.com/openeuler/openEuler-20.03-LTS-SP3/update/${OS_ARCH}/Packages/libbpf-0.3-4.oe1.${OS_ARCH}.rpm ${DOWNLOAD_DIR}
    gala_wget https://mirrors.aliyun.com/openeuler/openEuler-22.03-LTS-SP1/EPOL/main/${OS_ARCH}/Packages/flamegraph-1.0-1.oe2203sp1.noarch.rpm ${DOWNLOAD_DIR}
    gala_wget http://121.36.84.172/dailybuild/openEuler-20.03-LTS-SP1/openEuler-20.03-LTS-SP1/EPOL/main/${OS_ARCH}/Packages/cadvisor-0.37.0-2.oe1.${OS_ARCH}.rpm ${DOWNLOAD_DIR}
    gala_wget https://mirrors.aliyun.com/openeuler/openEuler-22.03-LTS-SP1/everything/${OS_ARCH}/Packages/cjson-1.7.15-1.oe2203sp1.${OS_ARCH}.rpm ${DOWNLOAD_DIR}
    gala_wget http://121.36.84.172/dailybuild/openEuler-20.03-LTS-SP1/openEuler-20.03-LTS-SP1/EPOL/main/${OS_ARCH}/Packages/python3-libconf-2.0.1-1.oe1.noarch.rpm ${DOWNLOAD_DIR}
}

download_gopher() {
    echo_info "- Download gala-gopher rpm"
    DOWNLOAD_DIR=$1

    if ! cat /etc/yum.conf | grep -q 'sslverify=false' ; then
        echo 'sslverify=false' >> /etc/yum.conf
    fi

    if [ "$OS_VERSION" == "openEuler-22.03-LTS-SP1" ] ; then
        yumdownloader --repofrompath="gala_eur,https://eur.openeuler.openatom.cn/results/Vchanger/gala-oe2203sp1/openeuler-22.03_LTS_SP1-${OS_ARCH}/" gala-gopher \
            --destdir=${DOWNLOAD_DIR} -b
        gopher_local_rpm=$(ls ${DOWNLOAD_DIR}/gala-gopher*oe2203sp1.*${OS_ARCH}.rpm)
        yum_download $gopher_local_rpm
    elif [ "$OS_VERSION" == "openEuler-22.03-LTS" ] ; then
        yumdownloader --repofrompath="gala_eur,https://eur.openeuler.openatom.cn/results/Vchanger/gala-oe2203/openeuler-22.03_LTS_SP1-${OS_ARCH}/" gala-gopher \
            --destdir=${DOWNLOAD_DIR} -b
        gopher_local_rpm=$(ls ${DOWNLOAD_DIR}/gala-gopher*oe2203.*${OS_ARCH}.rpm)
        yum_download $gopher_local_rpm
    elif [ "$OS_VERSION" == "openEuler-20.03-LTS-SP1" ] ; then
        yumdownloader --repofrompath="gala_eur,https://eur.openeuler.openatom.cn/results/Vchanger/gala-oe2003sp1/openeuler-20.03_LTS_SP3-${OS_ARCH}/" gala-gopher \
            --destdir=${DOWNLOAD_DIR} -b
        gopher_local_rpm=$(ls ${DOWNLOAD_DIR}/gala-gopher*oe1.*${OS_ARCH}.rpm)
        download_gopher_deps ${DOWNLOAD_DIR}
    elif [ "$OS_VERSION" == "kylin" ] ; then
        yumdownloader --repofrompath="gala_eur,https://eur.openeuler.openatom.cn/results/Vchanger/gala-kylin/openeuler-20.03_LTS_SP3-${OS_ARCH}/" gala-gopher \
            --destdir=${DOWNLOAD_DIR} -b
        gopher_local_rpm=$(ls ${DOWNLOAD_DIR}/gala-gopher*ky10.*${OS_ARCH}.rpm)
        download_gopher_deps ${DOWNLOAD_DIR}
    else
        echo_err_exit "Unsupported openEuler version, aborting!"
    fi
}

function deploy_gopher_rpm() {
    echo -e "[1] Installing gala-gopher"
    if [ "$DEPLOY_TYPE" == "local" ]; then
        install_rpm_local_repo gala-gopher
    else
        # mkdir -p ${WORKING_DIR}/gala-gopher-rpms
        # download_gopher ${WORKING_DIR}/gala-gopher-rpms
        if [ "$OS_VERSION" == "openEuler-22.03-LTS-SP1" ] || [ "$OS_VERSION" == "openEuler-22.03-LTS" ] ; then
            # install_rpm_remote_repo $gopher_local_rpm
            install_rpm_remote_repo gala-gopher
        else
            install_rpm log4cplus
            install_rpm python3-requests
            install_rpm python3-psycopg2
            install_rpm python3-yaml
            install_rpm librdkafka
            install_rpm libmicrohttpd
            yum install ${WORKING_DIR}/gala-gopher-rpms/python3-libconf-2.0.1-1.oe1.noarch.rpm \
                ${WORKING_DIR}/gala-gopher-rpms/cadvisor-0.37.0-2.oe1.${OS_ARCH}.rpm \
                ${WORKING_DIR}/gala-gopher-rpms/flamegraph-1.0-1.oe2203sp1.noarch.rpm \
                ${WORKING_DIR}/gala-gopher-rpms/libbpf-0.3-4.oe1.${OS_ARCH}.rpm \
                $gopher_local_rpm -y
        fi
    fi

    echo -e "\n[2] Configuring gala-gopher"
    # kafka broker
    sed -i "s#kafka_broker =.*#kafka_broker = \"${KAFKA_ADDR}\"#g" ${GOPHER_CONF}

    # pg_stat_probe.conf
    line=$(grep -n '  -' ${PG_STAT_CONF} | head -1 | cut -f1 -d':')
    sed -i "$((line+1)),\$d" ${PG_STAT_CONF}
    cat >> ${PG_STAT_CONF} << EOF
    ip: "172.11.0.101"
    port: "5432"
    dbname: "postgres"
    user: "opengauss_exporter"
    password: "opengauss_exporter123"
EOF

    # add guassdb to app whitelist
    if ! grep -q 'comm = "gaussdb"' ${GOPHER_APP_CONF} ; then
        sed -i "/^(/a\ \t{\n\t\tcomm = \"gaussdb\",\n\t\tcmdline = \"\";\n\t},"  ${GOPHER_APP_CONF}
    fi

    # stackprobe.conf
    sed -i "/name = \"stackprobe\"/{n;n;n;s/switch =.*/switch = \"on\"/g;}" ${GOPHER_CONF}
    sed -i "s/pyroscope_server.*/pyroscope_server = \"${PYROSCOPE_ADDR}\";/g" ${STACKPROBE_CONF}

    echo -e "\n[3] Starting gala-gopher service"
    systemctl restart gala-gopher || echo_err_exit "Error: fail to start gala-gopher.service"
}

function prepare_docker_gopher_conf() {
    mkdir -p /opt/gala/gopher_conf/extend_probes

    if [ "$DEPLOY_TYPE" != "local" ] ; then
        wget https://gitee.com/openeuler/gala-gopher/raw/master/config/gala-gopher.conf \
            -O /opt/gala/gopher_conf/gala-gopher.conf --no-check-certificate
        [ $? -ne 0 ] && echo_err_exit "Failed to download gala-gopher.conf"

        wget https://gitee.com/openeuler/gala-gopher/raw/master/config/gala-gopher-app.conf \
            -O /opt/gala/gopher_conf/gala-gopher-app.conf --no-check-certificate
        [ $? -ne 0 ] && echo_err_exit "Failed to download gala-gopher-app.conf"

        wget https://gitee.com/openeuler/gala-gopher/raw/master/src/probes/extends/ebpf.probe/src/stackprobe/conf/stackprobe.conf \
            -O /opt/gala/gopher_conf/extend_probes/stackprobe.conf --no-check-certificate
        [ $? -ne 0 ] && echo_err_exit "Failed to download gala-gopher stackprobe.conf"
    else
        \cp -f $LOCAL_DEPLOY_SRCDIR/gala-gopher.conf /opt/gala/gopher_conf/gala-gopher.conf
        \cp -f $LOCAL_DEPLOY_SRCDIR/gala-gopher-app.conf  /opt/gala/gopher_conf/gala-gopher-app.conf
        \cp -f $LOCAL_DEPLOY_SRCDIR/stackprobe.conf  /opt/gala/gopher_conf/extend_probes/stackprobe.conf
    fi
}

function deploy_gopher_docker() {
    container_name="gala-gopher"
    gopher_tag="gala-gopher-${OS_ARCH}:${GOPHER_DOCKER_TAG}"

    if [ "$DEPLOY_TYPE" == "local" ] ; then
        docker_load_image_file "$LOCAL_DEPLOY_SRCDIR/gala-gopher-${OS_ARCH}:${GOPHER_DOCKER_TAG}.tar"
    else
        echo -e "[1] Pulling/Loading gala-gopher docker image for ${GOPHER_DOCKER_TAG}"
        docker_pull_image "${gopher_tag}"
    fi

    echo -e "\n[2] Configuring gala-gopher"
    prepare_docker_gopher_conf
    # kafka broker
    sed -i "s#kafka_broker =.*#kafka_broker = \"${KAFKA_ADDR}\"#g" /opt/gala/gopher_conf/gala-gopher.conf

    # pg_stat_probe.conf
    cat > /opt/gala/gopher_conf/extend_probes/pg_stat_probe.conf << EOF
servers:
  -
    ip: "172.11.0.101"
    port: "5432"
    dbname: "postgres"
    user: "opengauss_exporter"
    password: "opengauss_exporter123"
EOF
    # add guassdb to app whitelist
    if ! grep -q 'comm = "gaussdb"' /opt/gala/gopher_conf/gala-gopher-app.conf ; then
        sed -i "/^(/a\ \t{\n\t\tcomm = \"gaussdb\",\n\t\tcmdline = \"\";\n\t}," /opt/gala/gopher_conf/gala-gopher-app.conf
    fi

    # stackprobe.conf
    sed -i "/name = \"stackprobe\"/{n;n;n;s/switch =.*/switch = \"on\"/g;}" /opt/gala/gopher_conf/gala-gopher.conf
    sed -i "s/pyroscope_server.*/pyroscope_server = \"${PYROSCOPE_ADDR}\";/g" /opt/gala/gopher_conf/extend_probes/stackprobe.conf

    echo -e "\n[3] Creating gala-gopher container"
    # Stop gala-gopher service to prevent port conflict
    systemctl stop gala-gopher 2>/dev/null
    docker stop ${container_name} 2>/dev/null ; docker rm ${container_name} 2>/dev/null
    docker run -d --name ${container_name} --privileged \
        -v /etc/os-release:/etc/os-release:ro -v /etc/localtime:/etc/localtime:ro \
        -v /sys:/sys  -v /boot:/boot:ro -v /usr/lib/debug:/usr/lib/debug \
        -v /var/run/docker.sock:/var/run/docker.sock -v /var/lib/docker:/var/lib/docker \
        -v /opt/gala/gopher_conf/:/gala-gopher/user_conf/ -v /:/host \
        --pid=host --network=host  ${DOCKER_HUB_TAG_PREFIX}/"${gopher_tag}"
    [ $? -ne 0 ] && echo_err_exit "Error: fail to run gala-gopher container"
}

function deploy_gopher() {
    echo_info "======Deploying gala-gopher(${GALA_DEPLOY_MODE})======"
    if [ "x${GALA_DEPLOY_MODE}" == "xrpm" ]  ; then
        deploy_gopher_rpm
    elif [ "x${GALA_DEPLOY_MODE}" == "xdocker" ] ; then
        deploy_gopher_docker
    else
        echo_err_exit "Unsupported deploy mode, must be rpm or docker"
    fi
    echo_info "======Deploying gala-gopher Done!======"
}


#=======Ops Deployment=======#
ANTEATER_CONF='/etc/gala-anteater/config/gala-anteater.yaml'
SPIDER_CONF='/etc/gala-spider/gala-spider.yaml'
INFERENCE_CONF='/etc/gala-inference/gala-inference.yaml'
ANTEATER_KAFKA_IP="localhost"
ANTEATER_PROMETHEUS_IP="localhost"
docker_anteater_conf=""
function parse_arg_ops() {
    ARGS=`getopt -a -o K:P:A:S: --long kafka:,prometheus:,arangodb:,docker,srcdir: -- "$@"`
    [ $? -ne 0 ] && (print_usage; exit 1)
    eval set -- "${ARGS}"
    while true
    do
        case $1 in
            -K|--kafka)
                KAFKA_ADDR=$(addr_add_port $2 ${KAFKA_PORT})
                KAFKA_PORT=$(get_port_from_addr ${KAFKA_ADDR})
                ANTEATER_KAFKA_IP=$(get_ip_from_addr ${KAFKA_ADDR})
                shift;;
            -P|--prometheus)
                PROMETHEUS_ADDR=$(addr_add_port $2 ${PROMETHEUS_PORT})
                PROMETHEUS_PORT=$(get_port_from_addr ${PROMETHEUS_ADDR})
                ANTEATER_PROMETHEUS_IP=$(get_ip_from_addr ${PROMETHEUS_ADDR})
                shift;;
            -A|--arangodb)
                ARANGODB_ADDR=$(addr_add_port $2 ${ARANGODB_PORT})
                shift;;
            -S|--srcdir)
                DEPLOY_TYPE="local"
                LOCAL_DEPLOY_SRCDIR=$(realpath $2)
                GALA_DEPLOY_MODE="docker"
                shift;;
            --docker)
                GALA_DEPLOY_MODE="docker"
                ;;
            --)
                shift
                break;;
            *)
                print_usage
                exit 1;;
        esac
        shift
    done
}

function deploy_ops_rpm() {
    echo -e "[1] Installing gala-ops"

    install_rpm gala-ops

    echo -e "\n[2] Configuring gala-ops"
    sed -i "/^Kafka:/{n;s/server:.*/server: \"${ANTEATER_KAFKA_IP}\"/g;}" ${ANTEATER_CONF}
    sed -i "/^Kafka:/{n;n;s/port:.*/port: \"${KAFKA_PORT}\"/g;}" ${ANTEATER_CONF}
    sed -i "/^Prometheus:/{n;s/server:.*/server: \"${ANTEATER_PROMETHEUS_IP}\"/g;}" ${ANTEATER_CONF}
    sed -i "/^Prometheus:/{n;n;s/port:.*/port: \"${PROMETHEUS_PORT}\"/g;}" ${ANTEATER_CONF}

    sed -i "/^prometheus:/{n;s/base_url:.*/base_url: \"http:\/\/${PROMETHEUS_ADDR}\/\"/g;}" ${SPIDER_CONF}
    sed -i "/^kafka:/{n;s/server:.*/server: \"${KAFKA_ADDR}\"/g;}" ${SPIDER_CONF}
    sed -i "/db_conf:/{n;s/url:.*/url: \"http:\/\/${ARANGODB_ADDR}\"/g;}" ${SPIDER_CONF}
    sed -i "s/  log_level:.*/  log_level: DEBUG/g" ${SPIDER_CONF}

    sed -i "/^prometheus:/{n;s/base_url:.*/base_url: \"http:\/\/${PROMETHEUS_ADDR}\/\"/g;}" ${INFERENCE_CONF}
    sed -i "/^kafka:/{n;s/server:.*/server: \"${KAFKA_ADDR}\"/g;}" ${INFERENCE_CONF}
    sed -i "/^arangodb:/{n;s/url:.*/url: \"http:\/\/${ARANGODB_ADDR}\"/g;}" ${INFERENCE_CONF}
    sed -i "s/  log_level:.*/  log_level: DEBUG/g" ${INFERENCE_CONF}

    echo -e "\n[3] Starting gala-ops service"
    systemctl restart gala-anteater || echo_err_exit "Error: fail to start gala-anteater service"
    systemctl restart gala-spider gala-inference || echo_err_exit "Error: fail to start gala-spider or gala-inference service"
}

function prepare_docker_anteater_config() {
    mkdir -p /opt/gala/anteater_conf
    if [ "$DEPLOY_TYPE" == "local" ] ; then
        [ ! -f "$LOCAL_DEPLOY_SRCDIR/gala-anteater.yaml" ] && echo_err_exit "Failed to find gala-anteater local yaml file"
        \cp -f $LOCAL_DEPLOY_SRCDIR/gala-anteater.yaml /opt/gala/anteater_conf/gala-anteater.yaml
    else
        wget https://gitee.com/openeuler/gala-anteater/raw/master/config/gala-anteater.yaml \
            -O $docker_anteater_conf --no-check-certificate
        [ $? -ne 0 ] && echo_err_exit "Failed to download gala-anteater.yaml"
    fi
}

function deploy_ops_docker() {
    echo -e "[1] Pulling gala-spider/gala-inference/gala-anteater docker image"
    spider_tag="gala-spider-${OS_ARCH}:1.0.1"
    infer_tag="gala-inference-${OS_ARCH}:1.0.1"
    anteater_tag="gala-anteater-${OS_ARCH}:1.0.1"

    if [ "$DEPLOY_TYPE" == "local" ] ; then
        docker_load_image_file "$LOCAL_DEPLOY_SRCDIR/gala-spider-${OS_ARCH}.tar"
        docker_load_image_file "$LOCAL_DEPLOY_SRCDIR/gala-inference-${OS_ARCH}.tar"
        docker_load_image_file "$LOCAL_DEPLOY_SRCDIR/gala-anteater-${OS_ARCH}.tar"
    elif [ "$DEPLOY_TYPE" == "remote" ] ; then
        docker_pull_image "${spider_tag}"
        docker_pull_image "${infer_tag}"
        docker_pull_image "${anteater_tag}"
    fi

    echo -e "\n[2] Creating gala-spider/gala-inference/gala-anteater container"
    docker stop gala-spider 2>/dev/null ; docker rm gala-spider 2>/dev/null
    docker run -d --name gala-spider \
        -e prometheus_server=${PROMETHEUS_ADDR} \
        -e arangodb_server=${ARANGODB_ADDR}  \
        -e kafka_server=${KAFKA_ADDR} \
        -e log_level=DEBUG --network host ${DOCKER_HUB_TAG_PREFIX}/"${spider_tag}"
    [ $? -ne 0 ] && echo_err_exit "Error: fail to run gala-spider container"

    docker stop gala-inference 2>/dev/null ; docker rm gala-inference 2>/dev/null
    docker run -d --name gala-inference \
        -e prometheus_server=${PROMETHEUS_ADDR} \
        -e arangodb_server=${ARANGODB_ADDR} \
        -e kafka_server=${KAFKA_ADDR} \
        -e log_level=DEBUG --network host ${DOCKER_HUB_TAG_PREFIX}/"${infer_tag}"
    [ $? -ne 0 ] && echo_err_exit "Error: fail to run gala-inference container"

    docker_anteater_conf="/opt/gala/anteater_conf/gala-anteater.yaml"
    prepare_docker_anteater_config
    sed -i "/^Kafka:/{n;s/server:.*/server: \"${ANTEATER_KAFKA_IP}\"/g;}" $docker_anteater_conf
    sed -i "/^Kafka:/{n;n;s/port:.*/port: \"${KAFKA_PORT}\"/g;}" $docker_anteater_conf
    sed -i "/^Prometheus:/{n;s/server:.*/server: \"${ANTEATER_PROMETHEUS_IP}\"/g;}" $docker_anteater_conf
    sed -i "/^Prometheus:/{n;n;s/port:.*/port: \"${PROMETHEUS_PORT}\"/g;}" $docker_anteater_conf

    docker stop gala-anteater 2>/dev/null ; docker rm gala-anteater 2>/dev/null
    docker run -d --name gala-anteater \
        -v $docker_anteater_conf:/etc/gala-anteater/config/gala-anteater.yaml \
        --network host ${DOCKER_HUB_TAG_PREFIX}/"${anteater_tag}"
    [ $? -ne 0 ] && echo_err_exit "Error: fail to run gala-anteater container"
}


function deploy_ops() {
    echo_info "======Deploying gala-ops(${GALA_DEPLOY_MODE})======"
    if [ "x${GALA_DEPLOY_MODE}" == "xrpm" ]  ; then
        if [ "$OFFICIAL_RELEASE" == "no" ] ; then
            echo_err_exit "gala-ops deployment in rpm mode is not supported on" $OS_VERSION
        fi
        deploy_ops_rpm
    elif [ "x${GALA_DEPLOY_MODE}" == "xdocker" ] ; then
        deploy_ops_docker
    else
        echo_err_exit "Unsupported deploy mode, must be rpm or docker"
    fi
    echo_info "======Deploying gala-ops Done!======"
}

#=======Middleware Deployment=======#
middleware_deploy_list=""
function parse_arg_middleware() {
    ARGS=`getopt -a -o K:P:E:S:Ap --long kafka:,prometheus:,elastic:,arangodb,pyroscope,srcdir: -- "$@"`
    [ $? -ne 0 ] && (print_usage; exit 1)
    eval set -- "${ARGS}"
    while true
    do
        case $1 in
            -K|--kafka)
                KAFKA_ADDR=$(addr_add_port "$2" ${KAFKA_PORT})
                middleware_deploy_list="${middleware_deploy_list}kafka "
                shift;;
            -P|--prometheus)
                PROMETHEUS_SCRAPE_LIST="$2"
                middleware_deploy_list="${middleware_deploy_list}prometheus "
                shift;;
            -E|--elastic)
                ES_ADDR=$(addr_add_port "$2" ${ES_PORT})
                middleware_deploy_list="${middleware_deploy_list}elasticsearch "
                shift;;
            -p|--pyroscope)
                middleware_deploy_list="${middleware_deploy_list}pyroscope "
                ;;
            -A|--arangodb)
                middleware_deploy_list="${middleware_deploy_list}arangodb "
                ;;
            -S|--srcdir)
                DEPLOY_TYPE="local"
                LOCAL_DEPLOY_SRCDIR=$(realpath $2)
                shift;;
            --)
                shift
                break;;
            *)
                print_usage
                exit 1;;
        esac
        shift
    done
}

KAFKA_VERSION='kafka_2.13-2.8.2'
KAFKA_WORKDIR="/opt/${KAFKA_VERSION}/"
function deploy_kafka() {
    echo -e "[-] Deploy kafka"
    echo -e "Installing..."
    if [ ! -d ${KAFKA_WORKDIR} ] ; then
        if ! which java > /dev/null 2>&1 ; then
            install_rpm java-1.8.0-openjdk
        fi

        if [ "$DEPLOY_TYPE" == "local" ] ; then
            KAFKA_LOCAL_TARBALL="$LOCAL_DEPLOY_SRCDIR/${KAFKA_VERSION}.tgz"
            [ ! -f "$KAFKA_LOCAL_TARBALL" ] && echo_err_exit "Error: fail to find kafka local tarball"
        else
            KAFKA_LOCAL_TARBALL="./${KAFKA_VERSION}.tgz"
            if [ ! -f "$KAFKA_LOCAL_TARBALL" ] ; then
                wget https://archive.apache.org/dist/kafka/2.8.2/${KAFKA_VERSION}.tgz --no-check-certificate
                [ $? -ne 0 ] && echo_err_exit "Error: fail to download kafka tarball from official website, check proxy!"
            fi
        fi
        tar xzf ${KAFKA_LOCAL_TARBALL} -C /opt
    fi

    echo -e "Configuring..."
    sed -i "0,/.*listeners=.*/s//listeners=PLAINTEXT:\/\/${KAFKA_ADDR}/" ${KAFKA_WORKDIR}/config/server.properties

    echo -e "Starting..."
    cd ${KAFKA_WORKDIR}
        ./bin/kafka-server-stop.sh
        ./bin/zookeeper-server-stop.sh
        ./bin/zookeeper-server-start.sh config/zookeeper.properties >/dev/null 2>&1 &
        i=0
        while ! netstat -tunpl | grep ':2181' | grep -q 'LISTEN' ; do
            sleep 5
            let i+=5
            if [ $i -ge 60 ] ; then
                echo_err_exit "Fail to start zookeeper, aborting"
            fi
        done
        ./bin/kafka-server-start.sh config/server.properties >/dev/null 2>&1 &
    cd - >/dev/null
}

NGINX_CONF='/etc/nginx/nginx.conf'
STATIC_SRC='/opt/PilotGo/agent/gala_deploy_middleware'
function deploy_nginx() {
    echo -e "[-] Deploy nginx"
    echo -e "Installing..."
    if ! rpm -qa | grep -q "nginx" 2>/dev/null ; then
        install_rpm nginx
    fi

    echo -e "Configuring..."
    \cp -f ${NGINX_CONF} "${NGINX_CONF}.bak"
    sed -i 's/user nginx/user root/g' ${NGINX_CONF}
    line=$(grep -n "# Settings for" ${NGINX_CONF} | cut -f1 -d':')
    sed -i "$line,\$d" ${NGINX_CONF}
    line=$(grep -n "server {" ${NGINX_CONF} | cut -f1 -d':')
    sed -i "$line,\$d" ${NGINX_CONF}

    cat >> ${NGINX_CONF} << EOF
    server {
        listen       ${NGINX_PORT};
        listen       ${NGINX_ADDR};
        server_name  _;
        root         /usr/share/nginx/html;

        # Load configuration files for the default server block.
        include /etc/nginx/default.d/*.conf;

        error_page 404 /404.html;
            location = /40x.html {
        }

        error_page 500 502 503 504 /50x.html;
            location = /50x.html {
        }

        location / {
        #alias /home/wjq/a-disk/aops-middleware;
        root $STATIC_SRC;
        expires max;
        autoindex off;
        }

    }
}

EOF

    echo -e "Starting..."
    systemctl restart nginx.service || echo_err_exit "Error: fail to start nginx.service"
}

PROMETHEUS_CONF='/etc/prometheus/prometheus.yml'
function deploy_prometheus2() {
    echo -e "[-] Deploy prometheus2"
    echo -e "Installing..."
    if ! rpm -qa | grep -q "prometheus2" 2>/dev/null ; then
        if [ "$DEPLOY_TYPE" == "local" ] ; then
            PROMETHEUS_LOCAL_RPM=$(ls $LOCAL_DEPLOY_SRCDIR/prometheus2*.${OS_ARCH}.rpm)
            [ ! -f "$PROMETHEUS_LOCAL_RPM" ] && echo_err_exit "Error: fail to find prometheus2 local rpm"
            yum install -y $PROMETHEUS_LOCAL_RPM
        else
            install_rpm prometheus2
        fi
    fi

    echo -e "Configuring..."
    \cp -f ${PROMETHEUS_CONF} "${PROMETHEUS_CONF}.bak"
    line=$(grep -n scrape_configs ${PROMETHEUS_CONF} | cut -f1 -d':')
    sed -i "$((line+1)),\$d" ${PROMETHEUS_CONF}
    scrape_array=(${PROMETHEUS_SCRAPE_LIST//,/ })

    for var in ${scrape_array[@]}
    do
        port=""
        job_name=${var%:*}
        if echo $var | grep -q ":" ; then
            port=${var##*:}
        fi
        port=${port:-8888}
        scrape_addr=${job_name##*-}

        cat >> ${PROMETHEUS_CONF} << EOF
  - job_name: "${job_name}"
    static_configs:
      - targets: ["${scrape_addr}:${port}"]

EOF
    done

    echo -e "Starting..."
    systemctl restart prometheus.service || echo_err_exit "Error: fail to start prometheus.service"
}

function deploy_arangodb_docker() {
    container_name="gala-arangodb"
    if [ $(getconf PAGE_SIZE) -gt 4096 ] ; then
        echo_err_exit "Arangodb not supported on systems whose PAGE SIZE larger than 4096"
    fi

    if docker inspect ${container_name} >/dev/null 2>&1 ; then
        echo -e "arangodb container has already been created, running"
        docker start ${container_name} || echo_err_exit "Error: fail to run arangodb container"
        return
    fi

    echo -e "Pulling/Loading arangodb docker images"
    arangodb_tag="arangodb-${OS_ARCH}"
    if [ "$DEPLOY_TYPE" == "local" ] ; then
        docker_load_image_file "$LOCAL_DEPLOY_SRCDIR/arangodb-${OS_ARCH}.tar"
    else
        docker_pull_image "${arangodb_tag}"
    fi

    echo -e "Creating and running arangodb container"
    docker run -d --name ${container_name} -p $ARANGODB_PORT:$ARANGODB_PORT -e ARANGO_NO_AUTH=yes ${DOCKER_HUB_TAG_PREFIX}/${arangodb_tag}
    [ $? -ne 0 ] && echo_err_exit "Error: fail to run arangodb container"
}

ARANGODB_CONF='/etc/arangodb3/arangod.conf'
function deploy_arangodb() {
    echo -e "[-] Deploy arangodb"
    if [ "${GALA_DEPLOY_MODE}" == "docker" ] || [ "$DEPLOY_TYPE" == "local" ] ; then
        deploy_arangodb_docker
        return
    fi

    if [ ${OS_ARCH} != 'x86_64' ] ; then
        echo_err_exit "Arangodb only available on x86_64 in rpm mode, try deploying with docker"
        deploy_arangodb_docker
        return
    fi

    if [ "$OFFICIAL_RELEASE" == "no" ] ; then
        echo_warn "arangodb deployment in rpm mode is not supported on $OS_VERSION, try deploying with docker"
        deploy_arangodb_docker
        return
    fi

    echo -e "Installing..."
    install_rpm arangodb3

    echo -e "Configuring..."
    sed -i 's/authentication =.*/authentication = false/g' ${ARANGODB_CONF}

    echo -e "Starting..."
    systemctl restart arangodb3.service || echo_err_exit "Error: fail to start arangodb3 service"
}

function deploy_elasticsearch() {
    echo_info "======Deploying Elasticsearch======"
    echo -e "[1] Downloading es tarball"

    if [ "$DEPLOY_TYPE" == "local" ] ; then
        ES_LOCAL_TARBALL="$LOCAL_DEPLOY_SRCDIR/elasticsearch-8.5.3-linux-${OS_ARCH}.tar.gz"
        [ ! -f "$ES_LOCAL_TARBALL" ] && echo_err_exit "Error: fail to find es local tarball"
    else
        ES_LOCAL_TARBALL="./elasticsearch-8.5.3-linux-${OS_ARCH}.tar.gz"
        if [ ! -f "$ES_LOCAL_TARBALL" ] ; then
            wget https://artifacts.elastic.co/downloads/elasticsearch/elasticsearch-8.5.3-linux-${OS_ARCH}.tar.gz --no-check-certificate
            [ $? -ne 0 ] && echo_err_exit "Error: fail to download elasticsearch rpm from official website, check proxy!"
        fi
    fi

    echo -e "\n[2] Creating elasticsearch-used user/group"
    groupadd elastic
    useradd -g elastic elastic
    \cp -f ${ES_LOCAL_TARBALL} /home/elastic
    chown elastic:elastic /home/elastic/elasticsearch-8.5.3-linux-${OS_ARCH}.tar.gz

    echo -e "\n[3] Starting es process"
    kill -9 $(ps -ef | grep elasticsearch-8.5.3 | awk '{if($3==1) print $2}')  2>/dev/null
    su - elastic -c "tar xzfm elasticsearch-8.5.3-linux-${OS_ARCH}.tar.gz && \
        cd elasticsearch-8.5.3 && \
        ES_JAVA_OPTS=\"-Xms1g -Xmx1g\" ./bin/elasticsearch -E xpack.security.enabled=false -E http.host=0.0.0.0 -d"
    echo_info "======Deploying Elasticsearch Done======"
    deploy_logstash
}

function deploy_logstash() {
    echo_info "======Deploying Logstash======"
    echo -e "[1] Downloading logstash rpm and install"

    if rpm -qa | grep -q logstash ; then
       echo "logstash is already installed, skip installing..."
    else
        LOGSTASH_LOCAL_RPM="$LOCAL_DEPLOY_SRCDIR/logstash-8.5.3-${OS_ARCH}.rpm"
        if [ "$DEPLOY_TYPE" == "local" ] ; then
            [ ! -f "$LOGSTASH_LOCAL_RPM" ] && echo_err_exit "Error: fail to find logstash local rpm"
        else
            LOGSTASH_LOCAL_RPM="./logstash-8.5.3-${OS_ARCH}.rpm"
            if [ ! -f "$LOGSTASH_LOCAL_RPM" ] ; then
                wget https://artifacts.elastic.co/downloads/logstash/logstash-8.5.3-${OS_ARCH}.rpm --no-check-certificate
                [ $? -ne 0 ] && echo_err_exit "Error: fail to download logstash rpm from official website, check proxy!"
            fi
        fi

        yum install ${LOGSTASH_LOCAL_RPM} -y || echo_err_exit "Error: fail to install $LOGSTASH_LOCAL_RPM"
    fi

    echo -e "\n[2] Configure logstash"
    rm -f /etc/logstash/logstash-sample.conf
    cat > /etc/logstash/conf.d/kafka2es.conf << EOF
input {
  kafka {
    bootstrap_servers => "${KAFKA_ADDR}"
    topics => ["gala_anteater_hybrid_model", "gala_cause_inference", "gala_gopher_event"]
    group_id => "hh_group"
    client_id => "hh_client"
    decorate_events => "true"
  }
}

filter {
  json {
    source => "message"
  }

  date {
    match => ["Timestamp", "UNIX_MS"]
    target => "@timestamp"
  }

  if [Attributes][event.name] {
    date {
      match => ["[Attributes][start_time]", "UNIX_MS"]
      target => "start_time"
    }

    date {
      match => ["[Attributes][end_time]", "UNIX_MS"]
      target => "end_time"
    }

    mutate {
      copy => { "[Attributes][event.name]" => "event_name" }
      copy => { "[Attributes][event.type]" => "event_type" }

      copy => { "[Resource][host.id]" => "host_id" }
      copy => { "[Resource][thread.pid]" => "pid" }
      copy => { "[Resource][thread.tgid]" => "tgid" }
      copy => { "[Resource][container.id]" => "container_id" }
      add_field => { "comm_pid" => "%{[Resource][thread.comm]}-%{[Resource][thread.pid]}" }
      add_field => { "comm_tgid" => "%{[Resource][process.name]}-%{[Resource][thread.tgid]}" }
      add_field => { "container_name_id" => "%{[Resource][container.name]}-%{[Resource][container.id]}" }
    }

    if ![event_type] or [event_type] == "" {
      mutate {
        add_field => { "event_type" => "other" }
      }
    }

    if [Attributes][func.stack] {
      mutate {
        gsub => [ "[Attributes][func.stack]", ";", "\n" ]
      }
    }

    if [event_type] == "file" {
      mutate {
        add_field => { "desc" => "EventName: %{[event_name]} ThreadName: \"%{[Resource][thread.comm]}\" Duration: %{[Attributes][duration]}ms FilePath: %{[Attributes][file.path]}" }
      }
    } else if [event_type] == "net" {
      mutate {
        add_field => { "desc" => "EventName: %{[event_name]} ThreadName: \"%{[Resource][thread.comm]}\" Duration: %{[Attributes][duration]}ms SockConn: %{[Attributes][sock.conn]}" }
      }
    } else if [event_type] == "futex" {
      mutate {
        add_field => { "desc" => "EventName: %{[event_name]} ThreadName: \"%{[Resource][thread.comm]}\" Duration: %{[Attributes][duration]}ms Operation: %{[Attributes][futex.op]}" }
      }
    } else if [event_type] == "oncpu" {
      mutate {
        add_field => { "desc" => "EventName: %{[event_name]} ThreadName: \"%{[Resource][thread.comm]}\" Duration: %{[Attributes][duration]}ms" }
      }
    } else {
      mutate {
        add_field => { "desc" => "EventName: %{[event_name]} ThreadName: \"%{[Resource][thread.comm]}\" Duration: %{[Attributes][duration]}ms" }
      }
    }
  }
}

output {
  elasticsearch {
    hosts => "${ES_ADDR}"
    index => "%{[@metadata][kafka][topic]}-%{+YYYY.MM.dd}"
  }
}
EOF

    echo -e "\n[3] Starting logstash service"
    cd /usr/share/logstash
    kill -9 $(ps -ef | grep logstash | grep 'kafka2es.conf' | awk '{print $2}') 2>/dev/null
    nohup ./bin/logstash -f  /etc/logstash/conf.d/kafka2es.conf &
    cd - > /dev/null
    echo_info "======Deploying Logstash Done======"
}

function deploy_pyroscope() {
    if ps -ef | grep -v grep | grep -q 'pyroscope server' ; then
        echo_info "pyroscope is already running, skip..."
        return
    fi

    if which pyroscope >/dev/null; then
        nohup pyroscope server &
        return
    fi

    if [ "$DEPLOY_TYPE" == "local" ] ; then
        PYROSCOPE_LOCAL_RPM="$LOCAL_DEPLOY_SRCDIR/pyroscope-0.37.2-1-${OS_ARCH}.rpm"
        [ ! -f "$PYROSCOPE_LOCAL_RPM" ] && echo_err_exit "Error: fail to find pyroscope local rpm"
    else
        PYROSCOPE_LOCAL_RPM="./pyroscope-0.37.2-1-${OS_ARCH}.rpm"
        if [ ! -f "$PYROSCOPE_LOCAL_RPM" ] ; then
            wget https://dl.pyroscope.io/release/pyroscope-0.37.2-1-${OS_ARCH}.rpm --no-check-certificate
            [ $? -ne 0 ] && echo_err_exit "Error: fail to download pyroscope rpm from official website, check proxy!"
        fi
    fi

    yum install ${PYROSCOPE_LOCAL_RPM} -y || echo_err_exit "Error: fail to install $PYROSCOPE_LOCAL_RPM"

    export PYROSCOPE_RETENTION=72h
    nohup pyroscope server &
}

function deploy_middleware() {
    echo_info "======Deploying MiddleWare======"
    [[ "${middleware_deploy_list}" =~ "kafka" ]] && deploy_kafka
    [[ "${middleware_deploy_list}" =~ "prometheus" ]] && deploy_prometheus2
    [[ "${middleware_deploy_list}" =~ "arangodb" ]] && deploy_arangodb
    [[ "${middleware_deploy_list}" =~ "elasticsearch" ]] && deploy_elasticsearch
    [[ "${middleware_deploy_list}" =~ "pyroscope" ]] && deploy_pyroscope

    echo_info "======Deploying MiddleWare Done!======"
}

#=======Grafana Deployment=======#
function parse_arg_grafana() {
    ARGS=`getopt -a -o P:p:E:A:S: --long prometheus:,pyroscope:,elastic:,arangodb:srcdir: -- "$@"`
    [ $? -ne 0 ] && ( print_usage; exit 1 )
    eval set -- "${ARGS}"
    while true
    do
        case $1 in
            -P|--prometheus)
                PROMETHEUS_ADDR=$(addr_add_port "$2" ${PROMETHEUS_PORT})
                shift;;
            -p|--pyroscope)
                PYROSCOPE_ADDR=$(addr_add_port "$2" ${PYROSCOPE_PORT})
                shift;;
            -E|--elastic)
                ES_ADDR=$(addr_add_port "$2" ${ES_PORT})
                shift;;
            -A|--arangodb)
                ARANGODB_ADDR=$(addr_add_port "$2" ${ARANGODB_PORT})
                shift;;
            -S|--srcdir)
                DEPLOY_TYPE="local"
                LOCAL_DEPLOY_SRCDIR=$(realpath $2)
                shift;;
            --)
                shift
                break;;
            *)
                print_usage
                exit 1;;
        esac
        shift
    done
}

function import_grafana_dashboard() {
    folder_uid="nErXDvCkzz"

    if [ "$DEPLOY_TYPE" == "local" ] ; then
        dashboard_path=${LOCAL_DEPLOY_SRCDIR}
    elif [ "$DEPLOY_TYPE" == "remote" ] ; then
        # wget dashboard json file
        download_grafana_dashboard ./
        dashboard_path="./"
    fi

    #1. get grafana token
    raw=`curl -X POST -H "Content-Type: application/json" -d '{"name":"apikeycurl3", "role":"Admin"}' \
         http://admin:admin@localhost:3000/api/auth/keys`
    id=`echo $raw | jq '.id'`
    token=`echo $raw | jq '.key'`
    token="${token#?}"
    token="${token%?}"
    auth="Authorization: Bearer "$token
    echo "id:"$id
    echo "token"$token

    #2. creat grafana folder
    body='{"uid":'\"$folder_uid\"', "title":"openEuler A-Ops"}'
    curl -X POST --insecure -H "${auth}" -H "Content-Type: application/json" \
        -d "$body" http://admin:admin@localhost:3000/api/folders

    #3. get data source list
    raw=`curl -X GET -H "Content-Type: application/json" -H "${auth}"  http://admin:admin@localhost:3000/api/datasources`
    namestr=`echo "$raw" | jq ' .[] | .name'`
    uidstr=`echo "$raw" | jq ' .[] | .uid'`
    typestr=`echo "$raw" | jq ' .[] | .type'`
    namelist=($namestr)
    uidlist=($uidstr)
    typelist=($typestr)

    inputs="["
    for i in "${!namelist[@]}"
    do
        ds_name=$(echo "DS_""${namelist[$i]}" | tr 'a-z' "A-Z"| sed 's/\"//g')
        str='{"name":"'$ds_name'", "type":"datasource", "pluginId":'${typelist[$i]}', "value":'${uidlist[$i]}'}'
        inputs=$inputs$str","
    done
    inputs=${inputs%?}"]"
    echo $inputs

    #4. import dashboard
    for ele in "${DASHBOARD_LIST[@]}"
    do
        json=$(cat "$dashboard_path/$ele")
        echo "import $dashboard_path/$ele"
        body='{"dashboard":'"$json"',
                "overwrite": true,
                "inputs": '"$inputs"',
                "folderUid":'\"$folder_uid\"'
                }'
        res=`curl -X POST --insecure -H "${auth}" -H "Content-Type: application/json" -d "${body}" http://localhost:3000/api/dashboards/import`
        if [ "$ele" == "A-Ops Home Page.json" ]; then
            home_page_uid=`echo $res | jq '.uid'`
        fi
    done

    #5. set grafana home page
    echo $home_page_id
    body='{"theme":"", "homeDashboardUID":'$home_page_uid',"timezone":"browser"}'
    curl -X PUT --insecure -H "${auth}" -H "Content-Type: application/json" -d "${body}" \
        http://localhost:3000/api/user/preferences

    #6. del grafana token
    curl -X DELETE -H "Content-Type: application/json" http://admin:admin@localhost:3000/api/auth/keys/$id
}


function deploy_grafana() {
    container_name="gala-grafana"

    echo_info "======Deploying Grafana======"

    if docker inspect ${container_name} >/dev/null 2>&1 ; then
        docker stop ${container_name}
        docker rm ${container_name}
    fi

    echo -e "\n[1] Pulling/Loading grafana docker image"
    if [ "$DEPLOY_TYPE" == "local" ] ; then
        docker_load_image_file "$LOCAL_DEPLOY_SRCDIR/grafana-${OS_ARCH}.tar"
    elif [ "$DEPLOY_TYPE" == "remote" ] ; then
        docker_pull_image "grafana-${OS_ARCH}"
    fi

    echo -e "\n[2] Creating grafana container"
    docker stop ${container_name} 2>/dev/null ; docker rm ${container_name} 2>/dev/null
    docker run -d --name ${container_name} --network host ${DOCKER_HUB_TAG_PREFIX}/grafana-${OS_ARCH}
    [ $? -ne 0 ] && echo_err_exit "Error: fail to run grafana container"

    echo -e "\n[3] Configuring datasources"
    i=0
    while ! netstat -tunpl | grep ':3000' | grep 'LISTEN' | grep -q 'grafana' ; do
        sleep 1
        let i+=1
        if [ $i -ge 10 ] ; then
            echo_err_exit "Fail to connect grafana, check container status"
        fi
    done

    name="Prometheus-dfs"
    result=$(curl -X POST -H "Content-Type: application/json" -d '{"name":"'${name}'","type":"prometheus",
"access":"proxy","url":"http://'${PROMETHEUS_ADDR}'","user":"","database":"",
"basicAuth":false,"isDefault":true,"jsonData":{"httpMethod":"POST"},"readOnly":false}' \
http://admin:admin@localhost:3000/api/datasources/ 2>/dev/null)
    if ! echo $result | grep -q 'Datasource added' ; then
        echo_err_exit "Fail to add ${name} datesource in grafana"
    fi

    name="pyroscope-datasource"
    result=$(curl -X POST -H "Content-Type: application/json" -d '{"name":"'${name}'","type":"pyroscope-datasource",
"access":"proxy","url":"","user":"","database":"","basicAuth":false,"isDefault":false,
"jsonData":{"path":"http://'${PYROSCOPE_ADDR}'"},"readOnly":false}' \
http://admin:admin@localhost:3000/api/datasources/ 2>/dev/null)
    if ! echo $result | grep -q 'Datasource added' ; then
        echo_err_exit "Fail to add ${name} datesource in grafana"
    fi

    name="Elasticsearch-anteater_hybrid_model"
    result=$(curl -X POST -H "Content-Type: application/json" -d '{"name":"'${name}'","type":"elasticsearch",
"access":"proxy","url":"http://'${ES_ADDR}'","user":"",
"database":"[gala_anteater_hybrid_model-]YYYY.MM.DD","basicAuth":false,"isDefault":false,
"jsonData":{"includeFrozen":false,"interval":"Daily","logLevelField":"","logMessageField":"","maxConcurrentShardRequests":5,"timeField":"@timestamp"},
"readOnly":false}' http://admin:admin@localhost:3000/api/datasources/ 2>/dev/null)
    if ! echo $result | grep -q 'Datasource added' ; then
        echo_err_exit "Fail to add ${name} datesource in grafana"
    fi

    name="Elasticsearch-cause_inference"
    result=$(curl -X POST -H "Content-Type: application/json" -d '{"name":"'${name}'","type":"elasticsearch",
"access":"proxy","url":"http://'${ES_ADDR}'","user":"",
"database":"[gala_cause_inference-]YYYY.MM.DD","basicAuth":false,"isDefault":false,
"jsonData":{"includeFrozen":false,"interval":"Daily","logLevelField":"","logMessageField":"","maxConcurrentShardRequests":5,"timeField":"@timestamp"},
"readOnly":false}' http://admin:admin@localhost:3000/api/datasources/ 2>/dev/null)
    if ! echo $result | grep -q 'Datasource added' ; then
        echo_err_exit "Fail to add ${name} datesource in grafana"
    fi

    name="Elasticsearch-gala-event"
    result=$(curl -X POST -H "Content-Type: application/json" -d '{"name":"'${name}'","type":"elasticsearch",
"access":"proxy","url":"http://'${ES_ADDR}'","user":"",
"database":"[gala_gopher_event-]YYYY.MM.DD","basicAuth":false,"isDefault":false,
"jsonData":{"includeFrozen":false,"interval":"Daily","logLevelField":"","logMessageField":"","maxConcurrentShardRequests":5,"timeField":"@timestamp"},
"readOnly":false}' http://admin:admin@localhost:3000/api/datasources/ 2>/dev/null)
    if ! echo $result | grep -q 'Datasource added' ; then
        echo_err_exit "Fail to add ${name} datesource in grafana"
    fi

    name="Elasticsearch-cause_inference_top"
    result=$(curl -X POST -H "Content-Type: application/json" -d '{"name":"'${name}'","type":"elasticsearch",
"access":"proxy","url":"http://'${ES_ADDR}'","user":"",
"database":"[gala_cause_inference-]YYYY.MM.DD","basicAuth":false,"isDefault":false,
"jsonData":{"includeFrozen":false,"interval":"Daily","logLevelField":"","logMessageField":"_source.Resource.top1","maxConcurrentShardRequests":5,"timeField":"@timestamp"},
"readOnly":false}' http://admin:admin@localhost:3000/api/datasources/ 2>/dev/null)
    if ! echo $result | grep -q 'Datasource added' ; then
        echo_err_exit "Fail to add ${name} datesource in grafana"
    fi

    name="Elasticsearch-graph"
    result=$(curl -X POST -H "Content-Type: application/json" -d '{"name":"'${name}'","type":"elasticsearch",
"access":"proxy","url":"http://'${ES_ADDR}'","user":"",
"database":"aops_graph2","basicAuth":false,"isDefault":false,
"jsonData":{"includeFrozen":false,"logLevelField":"","logMessageField":"_source","maxConcurrentShardRequests":5,"timeField":"timestamp"},
"readOnly":false}' http://admin:admin@localhost:3000/api/datasources/ 2>/dev/null)
    if ! echo $result | grep -q 'Datasource added' ; then
        echo_err_exit "Fail to add ${name} datesource in grafana"
    fi


    # Create topo graph es resources
    curl -X PUT "${ES_ADDR}/aops_graph2?pretty" >/dev/null 2>&1

    # Running daemon that transfrom arangodb to es
    [ ! -f ${WORKING_DIR}/arangodb2es.py ] && echo_err_exit "Failed to find arangodb2es.py"
    if ! which pip3 >/dev/null ; then
        if [ "$DEPLOY_TYPE" == "local" ] ; then
            local_pip_rpm=$(ls $LOCAL_DEPLOY_SRCDIR/python3-pip*.rpm)
            [ ! -f $local_pip_rpm ] && echo_err_exit "Error: failed to find local pip rpm"
            yum install -y $local_pip_rpm
        else
            install_rpm python3-pip
        fi
    fi

    if ! which jq >/dev/null ; then
        install_rpm jq
    fi

    if [ "$DEPLOY_TYPE" == "local" ] ; then
	    pushd $LOCAL_DEPLOY_SRCDIR
            pip3 install -q elasticsearch python-arango pytz pyArango --no-index --find-links=./
	    popd
    else
        pip3 install -q elasticsearch python-arango pytz pyArango -i http://mirrors.aliyun.com/pypi/simple/ --trusted-host mirrors.aliyun.com
    fi
    [ $? -ne 0 ] && echo_err_exit "Fail to pip install dependencies for arangodb2es.py"

    mkdir -p /opt/gala
    \cp  -f ${WORKING_DIR}/arangodb2es.py /opt/gala
    sed -i "s/self.arangodbUrl =.*/self.arangodbUrl = 'http:\/\/${ARANGODB_ADDR}'/g" /opt/gala/arangodb2es.py
    sed -i "s/self.esUrl =.*/self.esUrl = 'http:\/\/${ES_ADDR}'/g" /opt/gala/arangodb2es.py
    sed -i "s/self.promethusUrl =.*/self.promethusUrl = 'http:\/\/${PROMETHEUS_ADDR}'/g" /opt/gala/arangodb2es.py

    kill -9 $(ps -ef | grep arangodb2es.py | grep -v grep | awk '{print $2}')  2>/dev/null
    python3 /opt/gala/arangodb2es.py >/dev/null 2>&1 &

    echo -e "\n[4] import grafana dashboard"
    import_grafana_dashboard

    echo_info "======Deploying Grafana Done======"
}

#=======Main=======#
function detect_openEuler_version() {
    OS_VERSION=$(cat /etc/openEuler-latest | head -n1 | awk -F= '{print $2}' 2> /dev/null)
    if [ "$OS_VERSION" == "openEuler-22.03-LTS-SP1" ] ; then
        GOPHER_DOCKER_TAG="22.03-lts-sp1"
        REMOTE_REPO_PREFIX="$REMOTE_REPO_PREFIX/$OS_VERSION"
    elif [ "$OS_VERSION" == "openEuler-22.03-LTS" ] ; then
        GOPHER_DOCKER_TAG="22.03-lts"
        REMOTE_REPO_PREFIX="$REMOTE_REPO_PREFIX/openEuler-22.03-LTS-SP1"
        OFFICIAL_RELEASE="no"
    elif [ "$OS_VERSION" == "openEuler-20.03-LTS-SP1" ] ; then
        GOPHER_DOCKER_TAG="20.03-lts-sp1"
        REMOTE_REPO_PREFIX="$REMOTE_REPO_PREFIX/$OS_VERSION"
        OFFICIAL_RELEASE="no"
    else
        echo_err_exit "Unsupported openEuler version, aborting!"
    fi
}

function detect_os() {
    OS_TYPE=$(cat /etc/os-release | grep '^ID=' | awk -F '\"' '{print $2}')
    [ -z "$OS_TYPE" ] && echo_err_exit "Unsupport OS type, aborting!"

    if [ "x$OS_TYPE" == "xopenEuler" ] ; then
        detect_openEuler_version
    elif [ "x$OS_TYPE" == "xkylin" ] ; then
        [ ${OS_ARCH} != "x86_64" ] && echo_err_exit "Unsupported on Kylin aarch64"
        OS_VERSION="$OS_TYPE"
        REMOTE_REPO_PREFIX="$REMOTE_REPO_PREFIX/openEuler-20.03-LTS"
        OFFICIAL_RELEASE="no"
        GOPHER_DOCKER_TAG="kylin-v10"
    elif [ "x$OS_TYPE" == "xeuleros" ] ; then
        # TODO: support euleros
        OS_VERSION="$OS_TYPE"
        echo_err_exit "Unsupport OS type, aborting"
    else
        echo_err_exit "Unsupport OS type, aborting"
    fi
}


COMPONENT="$1"
shift

detect_os
case "x$COMPONENT" in
    xnginx)
        if [ ! -z "$2" ]; then
            NGINX_ADDR="${2}:${NGINX_PORT}"
        fi
        deploy_nginx
        ;;
    xgopher)
        parse_arg_gopher $@
        deploy_gopher
        ;;
    xops)
        parse_arg_ops $@
        deploy_ops
        ;;
    xopengauss)
        parse_arg_opengauss_server $@
        deploy_opengauss_server
        ;;
    xmiddleware)
        parse_arg_middleware $@
        deploy_middleware
        ;;
    xgrafana)
        parse_arg_grafana $@
        deploy_grafana
        ;;
    x)
        echo "Must specify a componet to be deployed!"
        print_usage
        exit 1
        ;;
    *)
        echo "Unsupport component:" $COMPONENT
        print_usage
        exit 1
        ;;
esac
