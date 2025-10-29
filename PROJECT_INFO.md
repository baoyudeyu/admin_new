# Telegram 群组管理机器人 - 项目信息

## 当前配置

- **Bot Token**: `8049083488:AAGeLQr1gQcu-uKzn1ZYJDLsak-hPlavSYQ`
- **作者 ID**: `7556123117`
- **数据库**: `admin_new@gz-cynosdbmysql-grp-jj6na063.sql.tencentcdb.com:21151`
- **通知频道**: 需要通过 `/config` 命令配置

## 数据库状态

✅ 已清理所有数据表：
- `authorized_groups` - 授权群组
- `global_admins` - 全局管理员
- `operation_logs` - 操作日志

其他表将在机器人首次启动时自动创建。

## 快速启动

### 1. 启动机器人
```bash
.\start.bat
```

### 2. 配置通知频道（必须）
- 私聊机器人发送 `/config`
- 选择 "📢 设置通知频道"
- 输入频道 ID

### 3. 添加授权群组
- 在 `/config` 中选择 "➕ 添加授权群组"
- 输入群组 ID

### 4. 在群组中测试
```
/jy @user 10s 测试
/help
```

## 重要提醒

### ⚠️ 隐私模式
如果机器人在公开群组中无法接收命令：
1. 找到 @BotFather
2. 发送 `/setprivacy`
3. 选择你的机器人
4. 选择 `Disable`

## 项目结构

```
admin/
├── admin-bot.exe          # 可执行文件
├── start.bat              # 启动脚本
├── README.md              # 项目说明
├── go.mod/go.sum          # Go 依赖
├── config/                # 配置文件
│   ├── config.yaml        # 主配置（已忽略）
│   └── config.example.yaml # 配置示例
├── cmd/bot/main.go        # 入口文件
└── internal/              # 核心代码
    ├── bot/               # 机器人逻辑
    ├── config/            # 配置管理
    ├── database/          # 数据库
    ├── models/            # 数据模型
    ├── scheduler/         # 定时任务
    ├── service/           # 业务服务
    └── utils/             # 工具函数
```

## 已修复的问题

### ✅ 命令响应问题
- 修复了授权群组命令无响应的问题
- 调整了命令处理顺序

### ✅ 显示格式优化
- ID 使用等宽格式显示
- 公开群组显示 @username
- 正确转义特殊字符

### ✅ 权限检查日志
- 添加了详细的权限检查日志
- 记录失败原因

### ✅ 通知频道配置
- 移除了硬编码的通知频道
- 改为通过 `/config` 动态配置

### ✅ 公开群组支持
- 识别并解决了隐私模式问题
- 添加了相关提示和文档

## 编译命令

```bash
# Windows
go build -o admin-bot.exe cmd/bot/main.go

# Linux
go build -o admin-bot cmd/bot/main.go
```

## 版本信息

- **最后更新**: 2025-10-27
- **Go 版本**: 1.21+
- **状态**: ✅ 生产就绪

---

**注意**: 此文件仅供项目维护使用，可以安全删除。



