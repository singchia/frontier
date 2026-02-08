# Frontier Systemd Service

这个目录包含了将Frontier作为systemd服务运行的配置文件。

## 文件说明

- `frontier.service` - systemd服务配置文件
- `install.sh` - 自动安装脚本
- `uninstall.sh` - 自动卸载脚本
- `README.md` - 本说明文件

## 快速安装

### 方法1: 使用Makefile（推荐）

```bash
# 使用Makefile安装systemd服务
sudo make install-systemd
```

### 方法2: 使用安装脚本

```bash
# 构建frontier二进制文件
make frontier

# 以root权限运行安装脚本
sudo ./dist/systemd/install.sh
```

### 方法3: 手动安装

1. 构建并安装frontier：
```bash
make install-frontier
```

2. 创建frontier用户：
```bash
sudo useradd --system --no-create-home --shell /bin/false frontier
```

3. 创建必要的目录：
```bash
sudo mkdir -p /var/log/frontier /var/lib/frontier
sudo chown -R frontier:frontier /var/log/frontier /var/lib/frontier
```

4. 安装systemd服务：
```bash
sudo cp dist/systemd/frontier.service /etc/systemd/system/
sudo systemctl daemon-reload
```

## 服务管理

### 启用和启动服务

```bash
# 启用服务（开机自启）
sudo systemctl enable frontier

# 启动服务
sudo systemctl start frontier

# 检查服务状态
sudo systemctl status frontier
```

### 服务控制

```bash
# 启动服务
sudo systemctl start frontier

# 停止服务
sudo systemctl stop frontier

# 重启服务
sudo systemctl restart frontier

# 重新加载配置
sudo systemctl reload frontier
```

### 查看日志

```bash
# 查看实时日志
sudo journalctl -u frontier -f

# 查看最近的日志
sudo journalctl -u frontier -n 100

# 查看特定时间段的日志
sudo journalctl -u frontier --since "2024-01-01" --until "2024-01-02"
```

## 配置说明

### 服务配置特性

- **用户隔离**: 以`frontier`用户运行，提高安全性
- **自动重启**: 服务异常退出时自动重启
- **资源限制**: 设置了文件描述符和进程数限制
- **安全设置**: 启用了多种安全保护措施
- **日志管理**: 输出到systemd journal

### 端口配置

Frontier默认监听以下端口：
- `30011` - Service bound端口
- `30012` - Edge bound端口

确保防火墙允许这些端口的访问。

### 配置文件

服务使用`/usr/conf/frontier.yaml`作为配置文件。你可以根据需要修改配置：

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

## 卸载

### 使用Makefile（推荐）

```bash
sudo make uninstall-systemd
```

### 使用卸载脚本

```bash
sudo ./dist/systemd/uninstall.sh
```

### 手动卸载

```bash
# 停止并禁用服务
sudo systemctl stop frontier
sudo systemctl disable frontier

# 删除服务文件
sudo rm /etc/systemd/system/frontier.service
sudo systemctl daemon-reload

# 删除用户（可选）
sudo userdel frontier

# 删除目录（可选）
sudo rm -rf /var/log/frontier /var/lib/frontier
```

## 故障排除

### 常见问题

1. **服务启动失败**
   - 检查二进制文件是否存在：`ls -la /usr/bin/frontier`
   - 检查配置文件是否存在：`ls -la /usr/conf/frontier.yaml`
   - 查看详细错误：`sudo journalctl -u frontier -n 50`

2. **端口被占用**
   - 检查端口使用情况：`sudo netstat -tlnp | grep -E ':(30011|30012)'`
   - 修改配置文件中的端口设置

3. **权限问题**
   - 确保frontier用户存在：`id frontier`
   - 检查目录权限：`ls -la /var/log/frontier /var/lib/frontier`

### 调试模式

如果需要调试，可以临时修改服务配置：

```bash
sudo systemctl edit frontier
```

添加以下内容：
```ini
[Service]
ExecStart=
ExecStart=/usr/bin/frontier --config /usr/conf/frontier.yaml -v 2
```

然后重启服务：
```bash
sudo systemctl restart frontier
```

## 安全注意事项

1. 服务以非特权用户运行
2. 启用了多种systemd安全特性
3. 建议定期更新frontier版本
4. 监控服务日志以发现异常活动
5. 确保配置文件权限正确（644）

## 支持

如果遇到问题，请：
1. 查看systemd日志：`journalctl -u frontier`
2. 检查frontier项目文档
3. 提交issue到项目仓库
