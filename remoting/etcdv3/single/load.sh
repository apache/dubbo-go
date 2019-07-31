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

source ./config.sh

idx=0
name="${name_prefix}${idx}"
ip="peer${idx}_ip"
client_port="peer${idx}_client_port"
peer_port="peer${idx}_peer_port"

opt=$1
switch $opt
