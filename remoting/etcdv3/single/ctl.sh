#!/usr/bin/env bash
# ******************************************************
# DESC    : etcd ctrl kv devops script
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : LGPL V3
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2018-01-08 23:32
# FILE    : ctl.sh
# ******************************************************

export ETCDCTL_API=3

peer0_ip="127.0.0.1"
peer0_client_port=2379

etcd_endpoints="http://${peer0_ip}:${peer0_client_port}"
ctl="etcdctl --endpoints=$etcd_endpoints"

usage() {
    echo "Usage: $0 set     key   value"
    echo "       $0 get     key"
    echo "       $0 get2    key  # output value with properties"
    echo "       $0 getj    key  # output value in json"
    echo "       $0 getr    [from  to)"
    echo "       $0 ls      dir" # get all info about key"
    echo "       $0 lr      dir"
    echo "       $0 rm      key"
    echo "       $0 rmr     dir"
    echo "       $0 watch   key"
    echo "       $0 watcha  dir"
    echo "       $0 lease   ID" # ID should be hex
    echo "       $0 status      # cluster status"
    exit
}

opt=$1
case C"$opt" in
    Cset)
        if [ $# != 3 ]; then
            usage
        fi
        $ctl put $2 $3 --prev-kv=true
        ;;
    Cget)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl --write-out="json" get $2 --order=ASCEND
        ;;
    Cget2)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl --write-out="fields" get $2 --order=ASCEND
        ;;
    Cgetj)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl --write-out="json" get $2 --order=ASCEND
        ;;
    Cgetr)
        if [ $# != 3 ]; then
            usage
        fi
        $ctl get "$2" "$3" --order=ASCEND
        ;;
    Clr)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl get $2 --order=DESCEND --prefix=true
        ;;
    Cls)
        dir=""
        if [ $# == 2 ]; then
            dir=$2
        fi
        $ctl get "$dir" --prefix=true
        ;;
    Crm)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl del $2 --prev-kv=true
        ;;
    Crmr)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl del $2 --prefix=true --prev-kv=true
        ;;
    Cwatch)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl watch $2 --prev-kv=true
        ;;
    Cwatcha)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl watch $2 --prev-kv=true --prefix=true
        ;;
    Clease)
        if [ $# != 2 ]; then
            usage
        fi
        $ctl lease timetolive $2
        ;;
    Cstatus)
        export ETCDCTL_API=2
        echo "-----------------member list----------------"
        $ctl member list
        echo "-----------------cluster health-------------"
        $ctl cluster-health
        export ETCDCTL_API=3
        ;;
    C*)
        usage
        ;;
esac

