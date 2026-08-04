#!/bin/bash
# EdgeOS postinstall script | EdgeOS 安装后脚本
set -e
echo "EdgeOS installed to /opt/edgeOS/"
chmod +x /opt/edgeOS/edgeOS

# 创建 systemd 服务 | Create systemd service
cat > /etc/systemd/system/edgeOS.service << 'UNIT'
[Unit]
Description=EdgeOS - Industrial Edge Agent Network Platform
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/edgeOS
ExecStart=/opt/edgeOS/edgeOS
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable edgeOS
echo "EdgeOS service created. Run 'systemctl start edgeOS' to start."
exit 0
