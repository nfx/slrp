#!/bin/bash

echo "[ + ] Running post-create script..."
echo "[ + ] Installing Node.js 20..."

# Check if we don't have nvm, if no - install it
echo "[ + ] Installing nvm..."
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.38.0/install.sh | bash
# Activate nvm
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

# Refresh bash
source ~/.bashrc

# Install node 20 
echo "[ + ] Installing Node.js 20..."
nvm install 20 && nvm use 20;

echo "[ + ] Installing UI dependencies and building..."
# Change to the "ui" directory
cd ui && npm install && cd ../

echo "[ + ] Installing Go dependencies and building..."
make build