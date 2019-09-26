#!/usr/bin/env sh

BASH_COMPLETION_DIR=$(pkg-config --variable=completionsdir bash-completion)
if [ -d ${BASH_COMPLETION_DIR} ]; then
    echo -e "==> Installing completion for Bash"
    /usr/bin/secrethub --completion-script-bash > ${BASH_COMPLETION_DIR}/secrethub
fi

if command -v zsh > /dev/null 2>&1; then
    echo -e "==> Installing completion for ZSH"
    mkdir -p ~/.zsh/completion
    /usr/local/bin/secrethub --completion-script-zsh > ~/.zsh/completion/secrethub

    if ! grep -q "source ~/.zsh/completion/secrethub" ~/.zshrc; then
        echo "source ~/.zsh/completion/secrethub" >> ~/.zshrc
    fi
fi
