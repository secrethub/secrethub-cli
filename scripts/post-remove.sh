#!/usr/bin/env sh

BASH_COMPLETION_DIR=$(pkg-config --variable=completionsdir bash-completion)
if [ -d ${BASH_COMPLETION_DIR} ]; then
    rm -f ${BASH_COMPLETION_DIR}/secrethub
fi
