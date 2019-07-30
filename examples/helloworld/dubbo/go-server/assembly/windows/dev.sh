#!/usr/bin/env bash
# ******************************************************
# DESC    : build script for dev env
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2018-06-24 17:34
# FILE    : dev.sh
# ******************************************************


set -e

export GOOS=windows
export GOARCH=amd64

PROFILE=dev

PROJECT_HOME=`pwd`

if [ -f "${PROJECT_HOME}/assembly/common/app.properties" ]; then
. ${PROJECT_HOME}/assembly/common/app.properties
fi


if [ -f "${PROJECT_HOME}/assembly/common/build.sh" ]; then
. ${PROJECT_HOME}/assembly/common/build.sh
fi
