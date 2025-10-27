# Telegram 群组管理机器人

一个功能强大的 Telegram 多群组管理机器人。

## 快速开始

### 1. 禁用机器人隐私模式（重要！）

**如果不执行此步骤，机器人在公开群组中将无法接收命令！**

1. 在 Telegram 中找到 `@BotFather`
2. 发送命令：`/setprivacy`
3. 选择你的机器人
4. 选择：`Disable`
5. 看到提示：`Success! The new status is: DISABLED.`

### 2. 配置

编辑 `config/config.yaml`：
```yaml
telegram:
  bot_token: "YOUR_BOT_TOKEN"
  author_id: YOUR_TELEGRAM_USER_ID
  notification_channel_id: 0  # 使用 /config 命令配置

database:
  host: "your-database-host"
  port: 3306
  username: "your-username"
  password: "your-password"
  database: "your-database"
```

### 3. 启动

Windows:
```bash
.\start.bat
```

Linux:
```bash
# 编译
go build -o admin-bot cmd/bot/main.go

# 运行
./admin-bot
```

### 4. 初始配置

私聊机器人发送 `/config`，进行以下配置：
1. 📢 设置通知频道（必须）
2. ➕ 添加授权群组
3. 👤 添加全局管理员（可选）

## 主要功能

### 管理命令
- `/t` - 踢出用户
- `/lh [时间] [理由]` - 拉黑用户
- `/unlh [理由]` - 解除拉黑
- `/jy [时间] [理由]` - 禁言用户
- `/unjy [理由]` - 解除禁言

### 使用方式
- 引用回复目标用户的消息
- 或在命令后指定 @username

### 时间单位
- `s` = 秒
- `m` = 分钟
- `h` = 小时
- `d` = 天

### 示例
```
/jy @user 10m 违规发言
/lh 1d 恶意刷屏
/t @spammer
```

## 项目结构

```
admin/
├── cmd/bot/main.go          # 主程序入口
├── config/                  # 配置文件
├── internal/
│   ├── bot/                 # 机器人核心逻辑
│   ├── config/              # 配置管理
│   ├── database/            # 数据库连接
│   ├── models/              # 数据模型
│   ├── scheduler/           # 定时任务
│   ├── service/             # 业务服务
│   └── utils/               # 工具函数
├── admin-bot.exe            # 编译后的可执行文件
└── start.bat                # 启动脚本

```

## 权限系统

1. **作者** - 最高权限
2. **全局管理员** - 可在所有授权群组执行管理操作
3. **群组管理员** - 仅可在当前群组执行管理操作

## 特性

- ✅ 多群组管理
- ✅ 批量操作支持
- ✅ 自动过期检查
- ✅ 黑名单同步
- ✅ 详细操作日志
- ✅ 通知频道推送
- ✅ 权限分级管理
- ✅ 未授权群组自动退出

## 常见问题

### Q: 机器人在公开群组中无法接收命令？

**A**: 这是因为 Telegram Bot 的隐私模式（Privacy Mode）默认启用。

**解决方法**：
1. 打开 Telegram，搜索 `@BotFather`
2. 发送：`/setprivacy`
3. 选择你的机器人
4. 选择：`Disable`
5. 重新添加机器人到群组

详细说明请查看：`PRIVACY_MODE_FIX.md`

### Q: 私密群组可以正常使用，公开群组不行？

**A**: 同上，需要禁用隐私模式。

### Q: 禁用隐私模式是否安全？

**A**: 对于群管机器人来说是安全的。代码已经正确过滤和处理消息。

## 技术栈

- Go 1.21+
- GORM (ORM)
- MySQL
- Telegram Bot API
- Viper (配置管理)
- Logrus (日志)
- Cron (定时任务)

## 编译

```bash
# Windows
go build -o admin-bot.exe cmd/bot/main.go

# Linux
go build -o admin-bot cmd/bot/main.go
```

## 许可证

MIT License

