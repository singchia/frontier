# Frontier Systemd Service

This directory contains files for running Frontier as a systemd service.

## Files

- `frontier.service` - systemd service unit file
- `install.sh` - automated installation script
- `uninstall.sh` - automated uninstallation script
- `README.md` - English documentation
- `README_cn.md` - Chinese documentation

## Quick Install

### Method 1: Makefile (Recommended)

```bash
# Install systemd service via Makefile
sudo make install-systemd
```

### Method 2: Install Script

```bash
# Build frontier binary
make frontier

# Run install script as root
sudo ./dist/systemd/install.sh
```

### Method 3: Manual Install

1. Build and install frontier:
```bash
make install-frontier
```

2. Create user:
```bash
sudo useradd --system --no-create-home --shell /bin/false frontier
```

3. Create required directories:
```bash
sudo mkdir -p /var/log/frontier /var/lib/frontier
sudo chown -R frontier:frontier /var/log/frontier /var/lib/frontier
```

4. Install service:
```bash
sudo cp dist/systemd/frontier.service /etc/systemd/system/
sudo systemctl daemon-reload
```

## Service Management

### Enable and Start

```bash
# Enable on boot
sudo systemctl enable frontier

# Start service
sudo systemctl start frontier

# Check status
sudo systemctl status frontier
```

### Control

```bash
# Start
sudo systemctl start frontier

# Stop
sudo systemctl stop frontier

# Restart
sudo systemctl restart frontier

# Reload config
sudo systemctl reload frontier
```

### Logs

```bash
# Follow logs
sudo journalctl -u frontier -f

# Last 100 lines
sudo journalctl -u frontier -n 100

# Time range
sudo journalctl -u frontier --since "2024-01-01" --until "2024-01-02"
```

## Configuration Notes

### Service Features

- **User isolation**: runs as dedicated `frontier` user
- **Auto restart**: restarts automatically on unexpected exits
- **Resource limits**: includes file descriptor and process limits
- **Hardening**: includes common systemd security settings
- **Logging**: logs to systemd journal

### Ports

Default listen ports:
- `30011` - Service bound
- `30012` - Edge bound

Make sure these ports are allowed by firewall rules.

### Config File

The service uses `/usr/conf/frontier.yaml` by default:

```yaml
edgebound:
  listen:
    network: tcp
    addr: 0.0.0.0:30012
  edgeid_alloc_when_no_idservice_on: true
servicebound:
  listen:
    network: tcp
    addr: 0.0.0.0:30011
```

## Uninstall

### Makefile (Recommended)

```bash
sudo make uninstall-systemd
```

### Script

```bash
sudo ./dist/systemd/uninstall.sh
```

### Manual

```bash
# Stop and disable service
sudo systemctl stop frontier
sudo systemctl disable frontier

# Remove service file
sudo rm /etc/systemd/system/frontier.service
sudo systemctl daemon-reload

# Remove user (optional)
sudo userdel frontier

# Remove directories (optional)
sudo rm -rf /var/log/frontier /var/lib/frontier
```

## Troubleshooting

### Common Issues

1. **Service failed to start**
   - Check binary: `ls -la /usr/bin/frontier`
   - Check config: `ls -la /usr/conf/frontier.yaml`
   - Check logs: `sudo journalctl -u frontier -n 50`

2. **Port conflicts**
   - Check ports: `sudo netstat -tlnp | grep -E ':(30011|30012)'`
   - Update listen ports in config

3. **Permission issues**
   - Verify user: `id frontier`
   - Check dir permissions: `ls -la /var/log/frontier /var/lib/frontier`

### Debug Mode

```bash
sudo systemctl edit frontier
```

Add:

```ini
[Service]
ExecStart=
ExecStart=/usr/bin/frontier --config /usr/conf/frontier.yaml -v 2
```

Then restart:

```bash
sudo systemctl restart frontier
```

## Security Notes

1. Run service as non-privileged user
2. Keep systemd hardening enabled
3. Update Frontier regularly
4. Monitor journal logs
5. Keep config file permissions strict

## Support

If you run into issues:
1. Check systemd logs: `journalctl -u frontier`
2. Check Frontier docs
3. Open an issue in the repository
