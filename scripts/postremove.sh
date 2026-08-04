#!/bin/bash
# EdgeOS postremove script | EdgeOS 卸载后脚本
set -e
echo "Removing EdgeOS service..."
systemctl disable edgeOS 2>/dev/null || true
rm -f /etc/systemd/system/edgeOS.service
systemctl daemon-reload
echo "EdgeOS removed. Data directory /opt/edgeOS/data retained."
exit 0
