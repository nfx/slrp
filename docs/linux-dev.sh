#!/bin/sh
sudo apt update

# setup nodejs 19.x
curl -fsSL https://deb.nodesource.com/setup_19.x | sudo -E bash -
sudo apt-get install -y nodejs

# setup go 1.19.x
GO_TGZ=go1.19.3.linux-amd64.tar.gz
wget https://go.dev/dl/$GO_TGZ
sudo tar -C /usr/local -xzf $GO_TGZ
rm $GO_TGZ
sudo sh -c 'echo "export PATH=$PATH:/usr/local/go/bin" >> /etc/profile'

# setup goreleaser
sudo snap install --classic goreleaser

mkdir $HOME/.slrp

tee $HOME/.slrp/config.yml <<EOF
app:
  state: $HOME/.slrp/data
  sync: 1m
log:
  level: info
  format: pretty
server:
  addr: "0.0.0.0:8089"
  read_timeout: 15s
mitm:
  addr: "0.0.0.0:8090"
  read_timeout: 15s
  idle_timeout: 15s
  write_timeout: 15s
EOF

# https://community.torproject.org/relay/setup/bridge/debian-ubuntu/
# See more: https://github.com/kevoreilly/CAPEv2/blob/master/installer/cape2.sh#L783-L802
sudo apt-get install tor -y