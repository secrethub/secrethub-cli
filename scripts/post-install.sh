#!/usr/bin/env sh

if [ -d /etc/bash_completion.d/ ]; then
    echo -e "==> Installing completion for Bash"
    /usr/local/bin/secrethub --completion-script-bash > /etc/bash_completion.d/secrethub
elif [ -d /usr/local/etc/bash_completion.d/ ]; then
    echo -e "==> Installing completion for Bash"
    /usr/local/bin/secrethub --completion-script-bash > /usr/local/etc/bash_completion.d/secrethub
fi

if command -v zsh > /dev/null 2>&1; then
    echo -e "==> Installing completion for ZSH"
    mkdir -p ~/.zsh/completion
    /usr/local/bin/secrethub --completion-script-zsh > ~/.zsh/completion/secrethub

    if ! grep -q "source ~/.zsh/completion/secrethub" ~/.zshrc; then
        echo "source ~/.zsh/completion/secrethub" >> ~/.zshrc
    fi
fi
