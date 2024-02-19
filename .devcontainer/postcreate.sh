#!/bin/bash

NVM_VERSION="0.38.0"
NODE_VERSION="20"

echo "[ + ] Running post-create script. Installing the following:"
echo "[ + ] NVM version: $NVM_VERSION"
echo "[ + ] NodeJS version: $NODE_VERSION"
echo "[ + ] =============================="

# Check if we don't have nvm, if no - install it
echo "[ + ] Installing nvm at v$NVM_VERSION"

curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v$NVM_VERSION/install.sh | bash
# Activate nvm
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

# Refresh bash
source ~/.bashrc

# Install node at a set versions
echo "[ + ] Installing NodeJS at v$NODE_VERSION"
nvm install $NODE_VERSION && nvm use $NODE_VERSION

echo "[ + ] Installing UI dependencies and building..."
# Change to the "ui" directory
cd ui && npm install && cd ../

echo "[ + ] Installing Go dependencies and building..."
make build
