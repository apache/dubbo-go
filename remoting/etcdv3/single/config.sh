#!/usr/bin/env bash
# ******************************************************
# DESC    : etcd devops script
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : LGPL V3
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2018-01-05 11:37
# FILE    : load.sh
# ******************************************************

cluster_name="etcd-test-cluster"
name_prefix="etcd_node"

data_dir=.
wal_dir=.
log_dir=.

peer0_ip="127.0.0.1"
peer0_client_port=2379
peer0_peer_port=2380

export ETCDCTL_API=3
etcd_endpoints="http://${peer0_ip}:${peer0_client_port}"
ctl="./etcdctl --endpoints=$etcd_endpoints"

usage() {
    echo "Usage: $0 start"
    echo "       $0 stop"
    echo "       $0 restart"
    echo "       $0 list"
    echo "       $0 clear"
    exit
}

start() {
    ./etcd --name=${name} \
        --data-dir=${data_dir} \
        --wal-dir=${wal_dir} \
        --initial-advertise-peer-urls http://${!ip}:${!peer_port} \
        --listen-peer-urls http://${!ip}:${!peer_port} \
        --listen-client-urls http://${!ip}:${!client_port},http://127.0.0.1:${!client_port} \
        --advertise-client-urls http://${!ip}:${!client_port} \
        --initial-cluster-token ${cluster_name} \
        --initial-cluster etcd_node0=http://${peer0_ip}:${peer0_peer_port} \
        --initial-cluster-state new  >> ${log_dir}/${name}.log 2>&1 &
    PID=`ps aux | grep -w  "name=${name}" | grep ${!client_port} | grep -v grep | awk '{print $2}'`
    if [ "$PID" != "" ];
    then
        for p in $PID
        do
            echo "start $name ( pid =" $p ")"
        done
    fi
}

term() {
    PID=`ps aux | grep -w "name=${name}" | grep ${!client_port} | grep -v grep | awk '{print $2}'`
    if [ "$PID" != "" ];
    then
        for ps in $PID
        do
            echo "kill -9 $name ( pid =" $ps ")"
        done
    fi
    kill -9 $ps
}

stop() {
    id=`${ctl} member list | grep ${name} | grep -v grep | awk -F ',' '{print $1;}'`
    echo "stop member name:${name} id:${id}"
    # echo "etcdctl member remove ${id}"
    # $ctl member remove ${id}
    term
}

list() {
    PID=`ps aux | grep -w "name=${name}" | grep ${!client_port} | grep -v grep | awk '{printf("%s,%s,%s,%s\n", $1, $2, $9, $10)}'`
    if [ "$PID" != "" ];
    then
        echo "list ${name} $role"
        echo "index: user, pid, start, duration"
        idx=0
        for ps in $PID
        do
            echo "$idx: $ps"
            ((idx ++))
        done
    fi
}

status() {
    echo "-----------------member list----------------"
    $ctl member list
    export ETCDCTL_API=2
    echo "-----------------cluster health-------------"
    $ctl cluster-health
    export ETCDCTL_API=3
}

client() {
    local host=$1
    local port=$2
    sh bin/zkCli.sh -server $host:$port
}

clear() {
    cd ${data_dir} && rm -rf ./* && cd ..
    cd ${wal_dir} && rm -rf ./* && cd ..
    cd ${log_dir} && rm -rf ./* && cd ..
}

switch() {
    opt=$1
    case C"$opt" in
        Cstart)
            start
            ;;
        Cstop)
            stop
            ;;
        Crestart)
            stop
            start
            ;;
        Clist)
            list
            ;;
        Cstatus)
            status
            ;;
        Cclear)
            clear
            ;;
        C*)
            usage
            ;;
    esac
}

