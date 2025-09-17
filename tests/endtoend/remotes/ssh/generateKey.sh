#!/bin/bash

OS=$1

#Generate Linux Keyfile
if [ "$OS" = "ubuntu-18.04" ]; then
  ssh-keygen -b 2048 -t rsa -f ./sshKey -q -N ""
fi

#Generate OSX Keyfile
if [ "$OS" = "macos-latest" ]; then
  ssh-keygen -b 2048 -t rsa -f ./sshKey -q -N "" <<<y 2>&1 >/dev/null
fi

#Generate Windows Keyfile (using WSL)
if [ "$OS" = "windows-latest" ] || [ "$OS" = "windows" ]; then
  # Check if WSL is available and use it, otherwise use native Windows ssh-keygen
  if command -v wsl &> /dev/null; then
    wsl ssh-keygen -b 2048 -t rsa -f ./sshKey -q -N ""
  else
    # Fallback to native Windows ssh-keygen if available
    ssh-keygen -b 2048 -t rsa -f ./sshKey -q -N ""
  fi
fi