# OminiGrid Bot

一个用 **Golang** 编写的网格交易 Telegram 机器人框架，用于连接多家 **Perp DEX**，并以可配置策略自动执行网格交易。

- 🐛 Bug 报告：[Issues](https://github.com/fachebot/omni-grid-bot/issues)
- 📧 使用交流：[电报群](https://t.me/+sRrZC-LVPAsyOWE1)

<img width="574" height="588" alt="Snipaste_2025-12-05_22-10-10" src="https://github.com/user-attachments/assets/431f1ddc-f7c1-4fdd-be13-24b79cc082ef" />

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

- **多交易所支持**

  - 当前已支持：**Lighter**、**Paradex**、**Variational**(前端 API)
  - 使用统一接口封装，便于后续扩展更多 Perp DEX

- **可配置的网格交易策略**

  - 通过 **Telegram Bot** 配置：
    - 网格间距、网格数量
    - 仓位大小、杠杆参数
    - 止盈/止损、风控阈值
  - 支持策略启停、参数动态调整

- **持久化与审计**

  - 使用 **SQLite3** 管理核心实体：
    - 订单、网格配置
    - 已成交记录、运行状态
  - 便于审计、回溯与策略复盘

- **异步事件与缓存**

  - 内部实现轻量级缓存与消息队列
  - 降低数据库读写压力，提高吞吐与响应速度

- **Telegram 告警与交互**
  - 通过电报机器人接收：
    - 策略运行状态、成交通知
    - 异常告警、风控触发
  - 支持部分管理指令（如暂停策略、查询状态等）

---

## 🚀 快速开始

下面以本地开发运行为例，演示如何在几分钟内跑起来。

> 📌 **普通用户提示**：如果您不是开发者，可以直接跳过下面的编译步骤，前往 [Release 页面](https://github.com/fachebot/omni-grid-bot/releases) 下载最新版本的可执行文件，然后直接查看 [配置说明](#⚙️-配置说明) 部分，修改配置文件后即可运行。

### 1. 环境准备

**先决条件：**

- 已安装 **Go 1.20+**
- 已安装 **Git**
- 已创建一个 Telegram Bot，并获取到 Bot Token  
  （可通过 [@BotFather](https://t.me/BotFather) 创建）

可选：

- 科学上网代理（若本地访问 DEX / Telegram 受限）

---

### 2. 克隆仓库

```bash
git clone https://github.com/fachebot/omni-grid-bot.git
cd omni-grid-bot
```

### 3. 配置文件

项目使用 `etc/config.yaml` 作为主配置文件，仓库中提供了一个样例，你可以手动复制 [etc/config.yaml.sampl](etc/config.yaml.sampl) 到 `etc/config.yaml` 文件，也可以执行下面的命令进行复制：

```bash
cp etc/config.yaml.sample etc/config.yaml
```

然后根据实际情况编辑 `etc/config.yaml`，主要包含：

- 代理服务器配置（如有需要）
- Telegram Bot 配置（如：Token、允许的管理用户 ID 等）

> 样例文件中的字段名尽量保持不变，仅修改值部分。

### 4. 构建与运行（开发环境）

```bash
go build

# linux
./omni-grid-bot

# windows
./omni-grid-bot.exe
```

运行后你可以：

- 观察控制台输出，确认是否成功连接交易所与 Telegram
- 查看 `logs/` 目录（如存在），了解详细运行日志

## ⚙️ 配置说明

当前配置较为简洁，核心集中在 `etc/config.yaml`：

```yml
# 代理服务器配置
Sock5Proxy:
  Host: 127.0.0.1 # 代理服务器地址
  Port: 10808 # 代理服务器端口
  Enable: false # 是否启用代理

# 电报机器人配置
TelegramBot:
  Debug: true
  ApiToken: 7916072799:AAFb-C25RgEAxNClxqeRpTkmO6C8e7FhzLs
  WhiteList: # 白名单列表，填写Telegram UserId(非白名单用户不允许使用机器人，如果白名单为空则所有人都可以使用)
    - 993021715
```

仓库内提供的 [etc/config.yaml.sample](etc/config.yaml.sample) 是最权威的字段参考，建议在阅读注释的基础上修改。

## 🕹 使用 Telegram 操作交易

程序启动成功后：

1. 打开与你创建的 Telegram 机器人的聊天对话框；
2. 输入 `/start` 命令开启主菜单；
3. 按照菜单提示创建新策略，然后设置交易所、交易对和网格参数，即可开始网格交易。

## 📁 项目结构（示例）

以下是一个简化的目录结构示意，便于快速了解代码布局：

```text
.
├── .vscode/             # VS Code 相关配置（可选）
├── data/                # 数据目录（如 SQLite 数据库、临时数据等）
├── etc/
│   ├── config.yaml      # 主配置文件（需用户创建）
│   └── config.yaml.sample
├── internal/            # 项目核心业务代码
│   ├── cache/           # 缓存实现，降低 DB / API 访问频率
│   ├── config/          # 配置加载与全局配置结构体
│   ├── engine/          # 策略执行引擎、网格调度的核心逻辑
│   ├── ent/             # ORM 实体定义、迁移等（数据库 schema 层）
│   ├── exchange/        # 各交易所适配层（Lighter/Paradex/Variational 等）
│   ├── helper/          # 交易所通用辅助方法、工具函数
│   ├── logger/          # 日志初始化与封装
│   ├── model/           # 封装对 ORM 实体的数据库操作函数（CRUD、查询组合等）
│   ├── strategy/        # 网格策略实现
│   ├── svc/             # 服务上下文
│   ├── telebot/         # Telegram Bot 交互层（指令解析、消息推送）
│   └── util/            # 其他通用工具（时间、错误处理、转换等）
├── logs/                # 日志输出目录（运行时生成）
├── main.go              # 程序入口
├── omni-grid-bot        # 已构建的二进制文件（Linux/macOS）
├── omni-grid-bot.exe    # 已构建的二进制文件（Windows）
├── go.mod
├── go.sum
├── LICENSE
└── README.md

```

## 🧪 开发与贡献

欢迎对以下方向进行改进或贡献：

- 新增交易所适配（新 Perp DEX）
- 新的网格变种策略 / 风控模块
- 更完善的回测 / 模拟执行能力
- 更丰富的 Telegram 管理指令与控制面板

建议工作流：

1. Fork 本仓库
2. 新建特性分支（如：feature/add-new-exchange）
3. 提交代码并确保通过基本构建
4. 发起 Pull Request，说明变更内容与动机

## ⚠️ 风险提示

本项目仅为技术研究与个人学习用途的网格交易框架示例：

- **不构成任何投资建议**
- 真实资金交易前，请在测试环境 / 小资金环境充分验证策略稳定性
- 使用过程中产生的任何资产损失由使用者自行承担

## 📜 许可证

本项目基于 MIT License 开源发布，详见 [LICENSE](LICENSE) 文件。
