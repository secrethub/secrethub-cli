#!/usr/bin/env sh

# Only execute if this is an uninstall, not an upgrade.
if [ "$1" = "0" ]; then
  CONF_DIR=$(pkg-config --variable=completionsdir bash-completion)
  if [ "${CONF_DIR}" != "" ]; then
    BASH_COMPLETION_DIR=$CONF_DIR
  elif [ -d /usr/share/bash-completion/completions/ ]; then
    BASH_COMPLETION_DIR=/usr/share/bash-completion/completions/
  fi

  if [ -d ${BASH_COMPLETION_DIR} ] && [ "${BASH_COMPLETION_DIR}" != "" ]; then
      rm -f ${BASH_COMPLETION_DIR}/secrethub
  fi
fi
