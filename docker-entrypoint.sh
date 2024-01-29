#!/bin/sh

_term() { 
  echo "Caught SIGTERM signal!" 
  kill -TERM "$child" 2>/dev/null
}

MITM_PORT=${MITM_PORT:-"8090"}
SERVER_PORT=${SERVER_PORT:-"8089"}

default_iface=$(awk '$2 == 00000000 { print $1 }' /proc/net/route)
listen_ip=$(ip addr show dev "$default_iface" | awk '$1 == "inet" { sub("/.*", "", $2); print $2 }')
echo "Chose IP: ${listen_ip}"
SLRP_MITM_ADDR="${listen_ip}:${MITM_PORT}" SLRP_SERVER_ADDR="${listen_ip}:${SERVER_PORT}" /opt/slrp

child=$! 
wait "$child"