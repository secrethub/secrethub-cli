#!/usr/bin/env sh

CONF_DIR=$(pkg-config --variable=completionsdir bash-completion 2> /dev/null) || true
if [ "${CONF_DIR}" != "" ]; then
  BASH_COMPLETION_DIR=$CONF_DIR
elif [ -d /usr/share/bash-completion/completions/ ]; then
  BASH_COMPLETION_DIR=/usr/share/bash-completion/completions/
fi

if [ "${BASH_COMPLETION_DIR}" != "" ] && [ -d ${BASH_COMPLETION_DIR} ]; then
    echo "Installing completion for Bash"
    /usr/bin/secrethub --completion-script-bash > ${BASH_COMPLETION_DIR}/secrethub
fi
