#!/usr/bin/env bash
# ******************************************************
# DESC    : dubbogo app devops script
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-05-13 02:01
# FILE    : load.sh
# ******************************************************

APP_NAME="APPLICATION_NAME"
APP_ARGS=""
SLEEP_INTERVAL=5
MAX_LIFETIME=4000

PROJECT_HOME=""
OS_NAME=`uname`
if [[ ${OS_NAME} != "Windows" ]]; then
    PROJECT_HOME=`pwd`
    PROJECT_HOME=${PROJECT_HOME}"/"
else
    APP_NAME="APPLICATION_NAME.exe"
fi

export APP_CONF_FILE=${PROJECT_HOME}"TARGET_CONF_FILE"
export APP_LOG_CONF_FILE=${PROJECT_HOME}"TARGET_LOG_CONF_FILE"
# export GOTRACEBACK=system
# export GODEBUG=gctrace=1

usage() {
    echo "Usage: $0 start"
    echo "       $0 stop"
    echo "       $0 term"
    echo "       $0 restart"
    echo "       $0 list"
    echo "       $0 monitor"
    echo "       $0 crontab"
    exit
}

start() {
    APP_LOG_PATH=${PROJECT_HOME}"logs/"
    mkdir -p ${APP_LOG_PATH}
    APP_BIN=${PROJECT_HOME}sbin/${APP_NAME}
    chmod u+x ${APP_BIN}
    # CMD="nohup ${APP_BIN} ${APP_ARGS} >>${APP_NAME}.nohup.out 2>&1 &"
    CMD="${APP_BIN}"
    eval ${CMD}
    PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $2}'`
    if [[ ${OS_NAME} != "Linux" && ${OS_NAME} != "Darwin" ]]; then
        PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $1}'`
    fi
    CUR=`date +%FT%T`
    if [ "${PID}" != "" ]; then
        for p in ${PID}
        do
            echo "start ${APP_NAME} ( pid =" ${p} ") at " ${CUR}
        done
    fi
}

stop() {
    PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $2}'`
    if [[ ${OS_NAME} != "Linux" && ${OS_NAME} != "Darwin" ]]; then
        PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $1}'`
    fi
    if [ "${PID}" != "" ];
    then
        for ps in ${PID}
        do
            echo "kill -SIGINT ${APP_NAME} ( pid =" ${ps} ")"
            kill -2 ${ps}
        done
    fi
}


term() {
    PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $2}'`
    if [[ ${OS_NAME} != "Linux" && ${OS_NAME} != "Darwin" ]]; then
        PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $1}'`
    fi
    if [ "${PID}" != "" ];
    then
        for ps in ${PID}
        do
            echo "kill -9 ${APP_NAME} ( pid =" ${ps} ")"
            kill -9 ${ps}
        done
    fi
}

list() {
    PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{printf("%s,%s,%s,%s\n", $1, $2, $9, $10)}'`
    if [[ ${OS_NAME} != "Linux" && ${OS_NAME} != "Darwin" ]]; then
        PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{printf("%s,%s,%s,%s,%s\n", $1, $4, $6, $7, $8)}'`
    fi

    if [ "${PID}" != "" ]; then
        echo "list ${APP_NAME}"

        if [[ ${OS_NAME} == "Linux" || ${OS_NAME} == "Darwin" ]]; then
            echo "index: user, pid, start, duration"
        else
            echo "index: PID, WINPID, UID, STIME, COMMAND"
        fi
        idx=0
        for ps in ${PID}
        do
            echo "${idx}: ${ps}"
            ((idx ++))
        done
    fi
}

monitor() {
    idx=0
    while true; do
        PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $2}'`
        if [[ ${OS_NAME} != "Linux" && ${OS_NAME} != "Darwin" ]]; then
            PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $1}'`
        fi
        if [[ "${PID}" == "" ]]; then
            start
            idx=0
        fi

        ((LIFE=idx*${SLEEP_INTERVAL}))
        echo "${APP_NAME} ( pid = " ${PID} ") has been working in normal state for " $LIFE " seconds."
        ((idx ++))
        sleep ${SLEEP_INTERVAL}
    done
}

crontab() {
    idx=0
    while true; do
        PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $2}'`
        if [[ ${OS_NAME} != "Linux" && ${OS_NAME} != "Darwin" ]]; then
            PID=`ps aux | grep -w ${APP_NAME} | grep -v grep | awk '{print $1}'`
        fi
        if [[ "${PID}" == "" ]]; then
            start
            idx=0
        fi

        ((LIFE=idx*${SLEEP_INTERVAL}))
        echo "${APP_NAME} ( pid = " ${PID} ") has been working in normal state for " $LIFE " seconds."
        ((idx ++))
        sleep ${SLEEP_INTERVAL}
        if [[ ${LIFE} -gt ${MAX_LIFETIME} ]]; then
            kill -9 ${PID}
        fi
    done
}

opt=$1
case C"$opt" in
    Cstart)
        start
        ;;
    Cstop)
        stop
        ;;
    Cterm)
        term
        ;;
    Crestart)
        term
        start
        ;;
    Clist)
        list
        ;;
    Cmonitor)
        monitor
        ;;
    Ccrontab)
        crontab
        ;;
    C*)
        usage
        ;;
esac

