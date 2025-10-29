# 部署指南

## 服务器部署步骤

### 1. 准备环境

```bash
# 安装 Go (如果未安装)
# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# CentOS/RHEL
sudo yum install golang

# 验证安装
go version
```

### 2. 上传代码

```bash
# 上传整个项目目录到服务器
# 使用 scp, rsync 或 git clone
```

### 3. 配置文件

```bash
# 复制配置模板
cp config/config.example.yaml config/config.yaml

# 编辑配置文件
vim config/config.yaml

# 必须配置:
# - telegram.bot_token
# - telegram.author_id
# - database.*
```

### 4. 编译程序

> **注意：** 程序已集成自动字符集修复功能，无需手动执行 SQL 脚本！
> 
> 程序启动时会自动检测并修复数据库字符集问题。
> 详见 `AUTO_CHARSET_FIX.md`

```bash
# 编译
go build -o admin-bot cmd/bot/main.go

# 或者使用优化编译
go build -ldflags="-s -w" -o admin-bot cmd/bot/main.go
```

### 5. 运行程序

```bash
# 直接运行
./admin-bot

# 或使用 nohup 后台运行
nohup ./admin-bot > bot.log 2>&1 &

# 或使用 screen
screen -S admin-bot
./admin-bot
# Ctrl+A+D 退出 screen

# 或使用 systemd (推荐)
# 参考下面的 systemd 配置
```

---

## Systemd 服务配置（推荐）

### 创建服务文件

```bash
sudo vim /etc/systemd/system/admin-bot.service
```

### 服务配置内容

```ini
[Unit]
Description=Telegram Admin Bot
After=network.target mysql.service

[Service]
Type=simple
User=your_username
WorkingDirectory=/path/to/admin
ExecStart=/path/to/admin/admin-bot
Restart=always
RestartSec=10
StandardOutput=append:/path/to/admin/bot.log
StandardError=append:/path/to/admin/bot.log

[Install]
WantedBy=multi-user.target
```

### 启用服务

```bash
# 重载 systemd
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start admin-bot

# 设置开机自启
sudo systemctl enable admin-bot

# 查看状态
sudo systemctl status admin-bot

# 查看日志
journalctl -u admin-bot -f
```

---

## 更新部署

```bash
# 1. 停止服务
sudo systemctl stop admin-bot

# 2. 备份旧版本
cp admin-bot admin-bot.backup

# 3. 拉取新代码
git pull
# 或重新上传代码

# 4. 重新编译
go build -o admin-bot cmd/bot/main.go

# 5. 启动服务
sudo systemctl start admin-bot

# 6. 检查状态
sudo systemctl status admin-bot
```

---

## 目录结构

```
admin/
├── cmd/
│   └── bot/
│       └── main.go          # 入口文件
├── config/
│   ├── config.example.yaml  # 配置模板
│   └── config.yaml          # 实际配置（不要提交到 git）
├── internal/                # 内部代码
├── scripts/
│   └── fix_charset.sql      # 数据库修复脚本
├── go.mod                   # Go 模块配置
├── go.sum                   # 依赖校验
├── PROJECT_INFO.md          # 项目说明
├── GROUP_DISPLAY_FIX.md     # 群组显示优化说明
└── README.md                # 项目文档
```

---

## 重要文件说明

### 必须保留
- `cmd/` - 入口程序
- `internal/` - 核心代码
- `config/` - 配置文件
- `go.mod`, `go.sum` - Go 依赖
- `scripts/fix_charset.sql` - 数据库修复脚本

### 不需要上传到服务器
- `admin-bot.exe` - Windows 可执行文件
- `.cursor/` - IDE 配置
- `*.log` - 日志文件

### 不要提交到 git
- `config/config.yaml` - 包含敏感信息
- `admin-bot` - 编译的可执行文件
- `*.log` - 日志文件

---

## 常见问题

### 1. 编译失败

```bash
# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod download

# 重新编译
go build -o admin-bot cmd/bot/main.go
```

### 2. 数据库连接失败

检查 `config/config.yaml` 中的数据库配置：
- host
- port
- username
- password
- database

### 3. 权限问题

```bash
# 给可执行文件添加执行权限
chmod +x admin-bot

# 如果使用 systemd，确保用户有权限
sudo chown your_username:your_group admin-bot
```

### 4. 端口占用

```bash
# 查看进程
ps aux | grep admin-bot

# 杀死进程
kill -9 <PID>
```

---

## 监控和日志

### 查看日志

```bash
# systemd 日志
journalctl -u admin-bot -f

# 或查看文件日志
tail -f bot.log
```

### 日志级别

在 `config/config.yaml` 中配置：
```yaml
system:
  log_level: "info"  # debug, info, warn, error
```

---

## 性能优化

### 编译优化

```bash
# 减小可执行文件大小
go build -ldflags="-s -w" -o admin-bot cmd/bot/main.go

# 或使用 upx 压缩
upx --best admin-bot
```

### 数据库优化

✅ **无需手动操作！**

程序启动时会自动检测并修复数据库字符集。

如需手动执行（可选）：
```bash
mysql -u user -p database < scripts/fix_charset.sql
```

---

## 安全建议

1. **配置文件权限**
   ```bash
   chmod 600 config/config.yaml
   ```

2. **使用专用用户运行**
   ```bash
   sudo useradd -r -s /bin/false admin-bot
   ```

3. **防火墙配置**
   - 确保服务器可以访问 Telegram API
   - 如果使用 Webhook，开放相应端口

4. **定期备份**
   - 备份数据库
   - 备份配置文件

---

## 快速部署脚本

```bash
#!/bin/bash

# 部署脚本
echo "开始部署..."

# 1. 编译
echo "正在编译..."
go build -o admin-bot cmd/bot/main.go

# 2. 停止旧服务
echo "停止旧服务..."
sudo systemctl stop admin-bot

# 3. 备份旧版本
if [ -f "admin-bot.old" ]; then
    rm admin-bot.old
fi
if [ -f "admin-bot" ]; then
    mv admin-bot admin-bot.old
fi

# 4. 移动新版本
mv admin-bot admin-bot

# 5. 设置权限
chmod +x admin-bot

# 6. 启动服务
echo "启动服务..."
sudo systemctl start admin-bot

# 7. 检查状态
echo "检查状态..."
sudo systemctl status admin-bot

echo "部署完成！"
```

---

**部署完成后，使用 /start 命令测试机器人是否正常运行。**

