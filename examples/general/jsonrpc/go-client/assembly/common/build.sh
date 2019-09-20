#!/usr/bin/env bash
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

rm -rf target/

PROJECT_HOME=`pwd`
TARGET_FOLDER=${PROJECT_HOME}/target/${GOOS}

TARGET_SBIN_NAME=${TARGET_EXEC_NAME}
version=`cat app/version.go | grep Version | grep -v "Apache"  | awk -F '=' '{print $2}' | awk -F '"' '{print $2}'`
if [[ ${GOOS} == "windows" ]]; then
    TARGET_SBIN_NAME=${TARGET_SBIN_NAME}.exe
fi
TARGET_NAME=${TARGET_FOLDER}/${TARGET_SBIN_NAME}
if [[ $PROFILE == "dev" ||  $PROFILE == "test" ]]; then
    # GFLAGS=-gcflags "-N -l" -race -x -v # -x会把go build的详细过程输出
    # GFLAGS=-gcflags "-N -l" -race -v
    # GFLAGS="-gcflags \"-N -l\" -v"
    cd ${BUILD_PACKAGE} && GOOS=$GOOS GOARCH=$GOARCH go build -gcflags "-N -l" -x -v -i -o ${TARGET_NAME} && cd -
else
    # -s去掉符号表（然后panic时候的stack trace就没有任何文件名/行号信息了，这个等价于普通C/C++程序被strip的效果），
    # -w去掉DWARF调试信息，得到的程序就不能用gdb调试了。-s和-w也可以分开使用，一般来说如果不打算用gdb调试，
    # -w基本没啥损失。-s的损失就有点大了。
    cd ${BUILD_PACKAGE} && GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-w" -x -v -i -o ${TARGET_NAME} && cd -
fi

TAR_NAME=${TARGET_EXEC_NAME}-${version}-`date "+%Y%m%d-%H%M"`-${PROFILE}

mkdir -p ${TARGET_FOLDER}/${TAR_NAME}

SBIN_DIR=${TARGET_FOLDER}/${TAR_NAME}/sbin
BIN_DIR=${TARGET_FOLDER}/${TAR_NAME}
CONF_DIR=${TARGET_FOLDER}/${TAR_NAME}/conf

mkdir -p ${SBIN_DIR}
mkdir -p ${CONF_DIR}

mv ${TARGET_NAME} ${SBIN_DIR}
cp -r assembly/bin ${BIN_DIR}
cd ${BIN_DIR}/bin/ && mv load.sh load_${TARGET_EXEC_NAME}.sh && cd -

platform=$(uname)
# modify APPLICATION_NAME
if [ ${platform} == "Darwin" ]; then
    sed -i "" "s~APPLICATION_NAME~${TARGET_EXEC_NAME}~g" ${BIN_DIR}/bin/*
else
    sed -i "s~APPLICATION_NAME~${TARGET_EXEC_NAME}~g" ${BIN_DIR}/bin/*
fi

# modify TARGET_CONF_FILE
if [ ${platform} == "Darwin" ]; then
    sed -i "" "s~TARGET_CONF_FILE~${TARGET_CONF_FILE}~g" ${BIN_DIR}/bin/*
else
    sed -i "s~TARGET_CONF_FILE~${TARGET_CONF_FILE}~g" ${BIN_DIR}/bin/*
fi

# modify TARGET_LOG_CONF_FILE
if [ ${platform} == "Darwin" ]; then
    sed -i "" "s~TARGET_LOG_CONF_FILE~${TARGET_LOG_CONF_FILE}~g" ${BIN_DIR}/bin/*
else
    sed -i "s~TARGET_LOG_CONF_FILE~${TARGET_LOG_CONF_FILE}~g" ${BIN_DIR}/bin/*
fi

cp -r profiles/${PROFILE}/* ${CONF_DIR}

cd ${TARGET_FOLDER}

tar czf ${TAR_NAME}.tar.gz ${TAR_NAME}/*

