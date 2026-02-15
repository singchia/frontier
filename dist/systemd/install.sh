#!/bin/bash

# Frontier Systemd Service Installation Script
# This script installs frontier as a systemd service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
FRONTIER_USER="frontier"
FRONTIER_GROUP="frontier"
BINARY_PATH="/usr/bin/frontier"
CONFIG_PATH="/usr/conf/frontier.yaml"
SERVICE_FILE="/etc/systemd/system/frontier.service"
LOG_DIR="/var/log/frontier"
DATA_DIR="/var/lib/frontier"

echo -e "${GREEN}Frontier Systemd Service Installation${NC}"
echo "========================================"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}This script must be run as root${NC}" 
   exit 1
fi

# Create frontier user and group
echo -e "${YELLOW}Creating frontier user and group...${NC}"
if ! id "$FRONTIER_USER" &>/dev/null; then
    useradd --system --no-create-home --shell /bin/false "$FRONTIER_USER"
    echo -e "${GREEN}Created user: $FRONTIER_USER${NC}"
else
    echo -e "${GREEN}User $FRONTIER_USER already exists${NC}"
fi

# Create directories
echo -e "${YELLOW}Creating directories...${NC}"
mkdir -p "$(dirname "$BINARY_PATH")"
mkdir -p "$(dirname "$CONFIG_PATH")"
mkdir -p "$LOG_DIR"
mkdir -p "$DATA_DIR"

# Set ownership
chown -R "$FRONTIER_USER:$FRONTIER_GROUP" "$LOG_DIR" "$DATA_DIR"
chmod 755 "$LOG_DIR" "$DATA_DIR"

echo -e "${GREEN}Created directories and set permissions${NC}"

# Install binary (if it exists)
if [[ -f "./bin/frontier" ]]; then
    echo -e "${YELLOW}Installing frontier binary...${NC}"
    cp "./bin/frontier" "$BINARY_PATH"
    chmod 755 "$BINARY_PATH"
    chown root:root "$BINARY_PATH"
    echo -e "${GREEN}Installed binary to $BINARY_PATH${NC}"
else
    echo -e "${YELLOW}Binary not found at ./bin/frontier, skipping binary installation${NC}"
    echo -e "${YELLOW}Please ensure frontier binary is installed at $BINARY_PATH${NC}"
fi

# Install config (if it exists)
if [[ -f "./etc/frontier.yaml" ]]; then
    echo -e "${YELLOW}Installing frontier configuration...${NC}"
    cp "./etc/frontier.yaml" "$CONFIG_PATH"
    chmod 644 "$CONFIG_PATH"
    chown root:root "$CONFIG_PATH"
    echo -e "${GREEN}Installed config to $CONFIG_PATH${NC}"
else
    echo -e "${YELLOW}Config not found at ./etc/frontier.yaml, skipping config installation${NC}"
    echo -e "${YELLOW}Please ensure frontier config is installed at $CONFIG_PATH${NC}"
fi

# Install systemd service
echo -e "${YELLOW}Installing systemd service...${NC}"
cp "./dist/systemd/frontier.service" "$SERVICE_FILE"
chmod 644 "$SERVICE_FILE"
chown root:root "$SERVICE_FILE"

# Reload systemd
systemctl daemon-reload

echo -e "${GREEN}Systemd service installed successfully!${NC}"
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo "1. Enable the service: systemctl enable frontier"
echo "2. Start the service: systemctl start frontier"
echo "3. Check status: systemctl status frontier"
echo "4. View logs: journalctl -u frontier -f"
echo ""
echo -e "${GREEN}Service management commands:${NC}"
echo "  Start:   systemctl start frontier"
echo "  Stop:    systemctl stop frontier"
echo "  Restart: systemctl restart frontier"
echo "  Status:  systemctl status frontier"
echo "  Logs:    journalctl -u frontier -f"
