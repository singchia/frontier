#!/bin/bash

# Frontier Systemd Service Uninstallation Script
# This script removes frontier systemd service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
FRONTIER_USER="frontier"
SERVICE_FILE="/etc/systemd/system/frontier.service"

echo -e "${GREEN}Frontier Systemd Service Uninstallation${NC}"
echo "=========================================="

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}This script must be run as root${NC}" 
   exit 1
fi

# Stop and disable service
echo -e "${YELLOW}Stopping and disabling frontier service...${NC}"
if systemctl is-active --quiet frontier; then
    systemctl stop frontier
    echo -e "${GREEN}Stopped frontier service${NC}"
else
    echo -e "${GREEN}Frontier service is not running${NC}"
fi

if systemctl is-enabled --quiet frontier; then
    systemctl disable frontier
    echo -e "${GREEN}Disabled frontier service${NC}"
else
    echo -e "${GREEN}Frontier service is not enabled${NC}"
fi

# Remove systemd service file
if [[ -f "$SERVICE_FILE" ]]; then
    rm -f "$SERVICE_FILE"
    echo -e "${GREEN}Removed systemd service file${NC}"
else
    echo -e "${GREEN}Systemd service file not found${NC}"
fi

# Reload systemd
systemctl daemon-reload

# Remove frontier user (optional)
read -p "Do you want to remove the frontier user? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if id "$FRONTIER_USER" &>/dev/null; then
        userdel "$FRONTIER_USER"
        echo -e "${GREEN}Removed frontier user${NC}"
    else
        echo -e "${GREEN}Frontier user does not exist${NC}"
    fi
fi

# Remove directories (optional)
read -p "Do you want to remove frontier directories (/var/log/frontier, /var/lib/frontier)? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf /var/log/frontier /var/lib/frontier
    echo -e "${GREEN}Removed frontier directories${NC}"
fi

# Remove binary and config (optional)
read -p "Do you want to remove frontier binary and config? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f /usr/bin/frontier /usr/conf/frontier.yaml
    echo -e "${GREEN}Removed frontier binary and config${NC}"
fi

echo -e "${GREEN}Frontier systemd service uninstalled successfully!${NC}"
