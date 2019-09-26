#!/usr/bin/env sh

BASH_COMPLETION_DIR=$(pkg-config --variable=completionsdir bash-completion)
if [ -d ${BASH_COMPLETION_DIR} ]; then
    echo -e "==> Installing completion for Bash"
    /usr/bin/secrethub --completion-script-bash > ${BASH_COMPLETION_DIR}/secrethub
fi
