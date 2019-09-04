#!/usr/bin/env sh

rm -f /etc/bash_completion.d/secrethub
rm -f /usr/local/etc/bash_completion.d/secrethub
rm -f ~/.zsh/completion/secrethub

sed -i "/source ~\/.zsh\/completion\/secrethub/d" ~/.zshrc > /dev/null 2>&1
sed -i "" "/source ~\/.zsh\/completion\/secrethub/d" ~/.zshrc > /dev/null 2>&1
