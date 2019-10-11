#!/usr/bin/env sh

# Only execute if this is an uninstall, not an upgrade.
if [ $1 -eq 0 ]; then
  BASH_COMPLETION_DIR=$(pkg-config --variable=completionsdir bash-completion)
  if [ -d ${BASH_COMPLETION_DIR} ] && [ ${BASH_COMPLETION_DIR} != "" ]; then
      rm -f ${BASH_COMPLETION_DIR}/secrethub
  fi
fi
