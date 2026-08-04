#!/bin/bash
# EdgeOS preremove script | EdgeOS 卸载前脚本
set -e
echo "Stopping EdgeOS service..."
systemctl stop edgeOS 2>/dev/null || true
exit 0
