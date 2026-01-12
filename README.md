# OminiGrid Bot

一个用 **Golang** 编写的网格交易 Telegram 机器人框架，用于连接多家 **Perp DEX**，并以可配置策略自动执行网格交易。

- 🐛 Bug 报告：[Issues](https://github.com/fachebot/omni-grid-bot/issues)
- 📧 使用交流：[电报群](https://t.me/+sRrZC-LVPAsyOWE1)

<img width="574" height="588" alt="Snipaste_2025-12-05_22-10-10" src="https://github.com/user-attachments/assets/431f1ddc-f7c1-4fdd-be13-24b79cc082ef" />

---

## 📋 目录

- [项目概览](#-项目概览)
- [主要特性](#-主要特性)
- [技术栈](#-技术栈)
- [快速开始](#-快速开始)
- [配置说明](#️-配置说明)
- [使用 Telegram 操作交易](#-使用-telegram-操作交易)
- [项目结构](#-项目结构)
- [开发与贡献](#-开发与贡献)
- [常见问题](#-常见问题)
- [风险提示](#️-风险提示)
- [许可证](#-许可证)

---

## 🧭 项目概览

OminiGrid Bot 专注于做一件事：  
**在多个永续合约 DEX 上，用统一的框架跑网格策略，并通过 Telegram 进行监控和管理。**

它主要提供：

- 与不同交易所的统一对接层
- 可配置、可扩展的网格策略管理
- 可靠的持久化与审计记录
- 通过 Telegram 进行通知与基本控制

非常适合用于：

- 搭建自用的网格交易机器人
- 尝试多交易所套利 / 多市场网格实验
- 作为量化交易基础框架进行二次开发

---

## ✨ 主要特性

### 多交易所支持

- 当前已支持：**Lighter**、**Paradex**、**Variational** (前端 API)
- 使用统一接口封装，便于后续扩展更多 Perp DEX
- 每个交易所独立 WebSocket 订阅，实时获取市场数据和订单状态

### 可配置的网格交易策略

- 通过 **Telegram Bot** 配置：
  - 网格间距、网格数量
  - 仓位大小、杠杆参数
  - 止盈/止损、风控阈值
- 支持策略启停、参数动态调整
- 支持多空两种网格模式

### 持久化与审计

- 使用 **SQLite3** 管理核心实体：
  - 订单、网格配置
  - 已成交记录、运行状态
- 使用 **Ent ORM** 进行数据库操作
- 便于审计、回溯与策略复盘

### 异步事件与缓存

- 内部实现轻量级缓存与消息队列
- 降低数据库读写压力，提高吞吐与响应速度
- 实时 WebSocket 事件处理

### Telegram 告警与交互

- 通过电报机器人接收：
  - 策略运行状态、成交通知
  - 异常告警、风控触发
  - 止盈/止损自动平仓通知
- 支持丰富的管理指令（创建策略、查询状态、修改参数等）
- 支持白名单机制，保障使用安全

---

## 🛠 技术栈

- **语言**: Go 1.20+
- **数据库**: SQLite3 (WAL 模式)
- **ORM**: Ent
- **WebSocket**: gorilla/websocket
- **Telegram Bot**: telebot.v4
- **日志**: logrus
- **配置**: YAML

---

## 🚀 快速开始

### 方式一：使用预编译版本（推荐普通用户）

1. 前往 [Release 页面](https://github.com/fachebot/omni-grid-bot/releases) 下载对应系统的可执行文件
2. 解压并进入目录
3. 参考 [配置说明](#️-配置说明) 配置 `etc/config.yaml`
4. 运行可执行文件

### 方式二：从源码编译

#### 1. 环境准备

**先决条件：**

- 已安装 **Go 1.20+**
- 已安装 **Git**
- 已创建一个 Telegram Bot，并获取到 Bot Token  
  （可通过 [@BotFather](https://t.me/BotFather) 创建）

**可选：**

- 科学上网代理（若本地访问 DEX / Telegram 受限）

#### 2. 克隆仓库

```bash
git clone https://github.com/fachebot/omni-grid-bot.git
cd omni-grid-bot
```

#### 3. 配置文件

项目使用 `etc/config.yaml` 作为主配置文件，仓库中提供了一个样例文件 `etc/config.yaml.sample`。

复制样例配置文件：

```bash
# Linux / macOS
cp etc/config.yaml.sample etc/config.yaml

# Windows
copy etc\config.yaml.sample etc\config.yaml
```

然后根据实际情况编辑 `etc/config.yaml`，主要配置项：

- 日志级别
- 代理服务器配置（如有需要）
- Telegram Bot 配置（Token、白名单用户 ID 等）

> 💡 **提示**：样例文件中的字段名尽量保持不变，仅修改值部分。

#### 4. 构建

```bash
go build
```

构建完成后会生成可执行文件：

- Linux/macOS: `./omni-grid-bot`
- Windows: `./omni-grid-bot.exe`

#### 5. 运行

```bash
# Linux / macOS
./omni-grid-bot

# Windows
./omni-grid-bot.exe

# 或指定配置文件路径
./omni-grid-bot -f /path/to/config.yaml
```

运行后你可以：

- 观察控制台输出，确认是否成功连接交易所与 Telegram
- 查看 `logs/` 目录（如存在），了解详细运行日志

---

## ⚙️ 配置说明

配置文件位于 `etc/config.yaml`，完整的配置示例请参考 [etc/config.yaml.sample](etc/config.yaml.sample)。

### 配置项说明

```yaml
# 日志设置
Log:
  Level: info  # 日志级别，可选值：trace, debug, info, warn, error

# 代理服务器配置
Sock5Proxy:
  Host: 127.0.0.1    # 代理服务器地址
  Port: 10808        # 代理服务器端口
  Enable: false      # 是否启用代理

# 电报机器人配置
TelegramBot:
  Debug: false                           # 调试模式
  ApiToken: YOUR_BOT_TOKEN_HERE         # Telegram Bot Token (从 @BotFather 获取)
  WhiteList:                            # 白名单列表（Telegram User ID）
    - 123456789                          # 如果列表为空，则所有人都可以使用
  NotifyChatId: 0                       # 可选：通知聊天 ID（用于发送系统通知）
```

### 配置详解

#### Log 配置

- `Level`: 日志级别，从低到高依次为：`trace` < `debug` < `info` < `warn` < `error`
  - 生产环境建议使用 `info` 或 `warn`
  - 调试时可以使用 `debug` 或 `trace`

#### Sock5Proxy 配置

如果您的网络环境无法直接访问 DEX 或 Telegram，可以配置 SOCKS5 代理。

- `Host`: 代理服务器地址
- `Port`: 代理服务器端口
- `Enable`: 是否启用代理（`true`/`false`）

#### TelegramBot 配置

- `Debug`: 是否启用调试模式
- `ApiToken`: Telegram Bot Token，从 [@BotFather](https://t.me/BotFather) 创建 Bot 后获取
- `WhiteList`: 白名单用户 ID 列表
  - 如果列表为空（`[]`），则所有用户都可以使用机器人
  - 如果列表不为空，则只有列表中的用户 ID 可以使用机器人
  - 获取用户 ID：可以通过 [@userinfobot](https://t.me/userinfobot) 获取
- `NotifyChatId`: 可选，用于发送系统通知的聊天 ID（如群组 ID）

> ⚠️ **安全提示**：请妥善保管您的 `ApiToken`，不要将其提交到公共代码仓库。

---

## 🕹 使用 Telegram 操作交易

程序启动成功后：

1. 打开与你创建的 Telegram 机器人的聊天对话框
2. 输入 `/start` 命令开启主菜单
3. 按照菜单提示操作：
   - **创建策略**：配置交易所、交易对和网格参数
   - **查看策略列表**：查看所有已创建的策略
   - **策略管理**：启停策略、修改参数、查看详情
   - **交易所设置**：配置各交易所的账户信息
   - **查看成交记录**：查看策略的历史成交情况

### 主要功能

- ✅ 创建和管理网格交易策略
- ✅ 实时查看策略运行状态
- ✅ 动态调整策略参数
- ✅ 查看成交记录和盈亏情况
- ✅ 接收交易通知和告警信息
- ✅ 手动平仓操作

---

## 📁 项目结构

```
.
├── cmd/                  # 命令行工具和测试代码
├── data/                 # 数据目录（SQLite 数据库、临时数据等）
│   └── sqlite.db        # SQLite 数据库文件（运行时自动创建）
├── etc/                  # 配置文件目录
│   ├── config.yaml       # 主配置文件（需用户创建）
│   └── config.yaml.sample # 配置文件样例
├── internal/             # 项目核心业务代码
│   ├── cache/            # 缓存实现，降低 DB / API 访问频率
│   ├── config/           # 配置加载与全局配置结构体
│   ├── engine/           # 策略执行引擎、网格调度的核心逻辑
│   ├── ent/              # ORM 实体定义、迁移等（数据库 schema 层）
│   ├── exchange/         # 各交易所适配层（Lighter/Paradex/Variational 等）
│   ├── helper/           # 交易所通用辅助方法、工具函数
│   ├── logger/           # 日志初始化与封装
│   ├── model/            # 封装对 ORM 实体的数据库操作函数（CRUD、查询组合等）
│   ├── strategy/         # 网格策略实现
│   ├── svc/              # 服务上下文
│   ├── telebot/          # Telegram Bot 交互层（指令解析、消息推送）
│   └── util/             # 其他通用工具（时间、错误处理、转换等）
├── logs/                 # 日志输出目录（运行时生成）
├── main.go               # 程序入口
├── go.mod                # Go 模块定义
├── go.sum                # Go 模块校验和
├── LICENSE               # 许可证文件
└── README.md             # 本文件
```

---

## 🧪 开发与贡献

欢迎对以下方向进行改进或贡献：

- 新增交易所适配（新 Perp DEX）
- 新的网格变种策略 / 风控模块
- 更完善的回测 / 模拟执行能力
- 更丰富的 Telegram 管理指令与控制面板
- 性能优化和 Bug 修复
- 文档改进

### 贡献流程

1. Fork 本仓库
2. 新建特性分支（如：`feature/add-new-exchange`）
3. 提交代码并确保通过基本构建
4. 编写或更新相关文档
5. 发起 Pull Request，详细说明变更内容与动机

### 开发环境

```bash
# 克隆仓库
git clone https://github.com/fachebot/omni-grid-bot.git
cd omni-grid-bot

# 安装依赖
go mod download

# 运行测试（如有）
go test ./...

# 构建
go build
```

---

## ❓ 常见问题

### Q: 如何获取 Telegram User ID？

A: 可以通过以下方式获取：
- 使用 [@userinfobot](https://t.me/userinfobot) 机器人，发送任意消息即可获取您的 User ID
- 或使用 [@getidsbot](https://t.me/getidsbot) 获取

### Q: 如何创建 Telegram Bot？

A: 
1. 在 Telegram 中搜索 [@BotFather](https://t.me/BotFather)
2. 发送 `/newbot` 命令
3. 按照提示设置 Bot 名称和用户名
4. 创建成功后，BotFather 会返回 Bot Token，将其填入配置文件

### Q: 数据库文件在哪里？

A: 数据库文件默认位于 `data/sqlite.db`，程序首次运行时会自动创建。如需备份，直接复制该文件即可。

### Q: 如何查看详细的运行日志？

A: 
- 日志默认输出到控制台
- 如需查看文件日志，请检查 `logs/` 目录
- 在配置文件中设置 `Log.Level: debug` 可以看到更详细的调试信息

### Q: 支持哪些交易所？

A: 当前已支持：
- **Lighter** - 去中心化永续合约交易所
- **Paradex** - Starknet 上的永续合约 DEX
- **Variational** - 使用前端 API 对接

### Q: 如何添加新的交易所支持？

A: 参考 `internal/exchange/` 目录下现有交易所的实现，实现统一的交易所接口即可。详细说明请参考代码注释和现有实现。

### Q: 策略数据会丢失吗？

A: 所有策略配置和交易记录都存储在 SQLite 数据库中，程序重启后会自动加载所有活跃策略。只要数据库文件未损坏，数据不会丢失。

### Q: 程序崩溃后如何恢复？

A: 程序重启后会自动加载所有活跃策略并恢复运行。如果遇到异常，请检查日志文件了解具体错误信息。

---

## ⚠️ 风险提示

本项目仅为技术研究与个人学习用途的网格交易框架示例：

- **不构成任何投资建议**
- 真实资金交易前，请在测试环境 / 小资金环境充分验证策略稳定性
- 使用过程中产生的任何资产损失由使用者自行承担
- 请妥善保管您的私钥和 API 密钥，不要泄露给他人
- 建议在充分了解网格交易策略的风险后再使用

---

## 📜 许可证

本项目基于 MIT License 开源发布，详见 [LICENSE](LICENSE) 文件。
