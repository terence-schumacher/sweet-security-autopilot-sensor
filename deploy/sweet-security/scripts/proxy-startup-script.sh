#!/usr/bin/env bash
set -e

# Sweet IP
TARGET_IP=18.220.208.31

# Get the local IP address
LOCAL_IP=$(hostname -I | awk '{print $1}')
TARGET_PORT=443
LOCAL_PORT=443

# Enable IPv4 forwarding
sysctl -w net.ipv4.ip_forward=1 > /dev/null
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf

# Wait for network to be fully up
sleep 5

# Clear existing NAT rules
iptables -t nat -F
iptables -t nat -X

# Set up DNAT: Traffic arriving at LOCAL_IP:LOCAL_PORT will be forwarded to TARGET_IP:TARGET_PORT
iptables -t nat -A PREROUTING -p tcp -d $LOCAL_IP --dport $LOCAL_PORT -j DNAT --to-destination $TARGET_IP:$TARGET_PORT
iptables -t nat -A PREROUTING -p udp -d $LOCAL_IP --dport $LOCAL_PORT -j DNAT --to-destination $TARGET_IP:$TARGET_PORT

# Set up MASQUERADE so that return traffic passes back through this host
iptables -t nat -A POSTROUTING -p tcp -d $TARGET_IP --dport $TARGET_PORT -j MASQUERADE
iptables -t nat -A POSTROUTING -p udp -d $TARGET_IP --dport $TARGET_PORT -j MASQUERADE

# Ensure FORWARD chain rules allow the forwarded traffic
iptables -A FORWARD -p tcp -d $TARGET_IP --dport $TARGET_PORT -j ACCEPT
iptables -A FORWARD -p tcp -s $TARGET_IP --sport $TARGET_PORT -j ACCEPT
iptables -A FORWARD -p udp -d $TARGET_IP --dport $TARGET_PORT -j ACCEPT
iptables -A FORWARD -p udp -s $TARGET_IP --sport $TARGET_PORT -j ACCEPT

# Save rules (Ubuntu/Debian)
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y iptables-persistent
iptables-save > /etc/iptables/rules.v4
