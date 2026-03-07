![gstor](https://user-images.githubusercontent.com/21356580/216275709-3ed3420b-5c35-47d7-b824-582bcf933c4d.jpeg)
Gstor 是一个应用于 Linux 服务器上的硬盘管理工具。

![https://github.com/serialq7ic4/gstor/actions/workflows/test.yml](https://github.com/serialq7ic4/gstor/workflows/test/badge.svg)

# 概述

Gstor 是一个使用 Cobra 构建的命令行工具，面向 Linux 服务器的磁盘巡检、定位和基础管理场景。

项目当前通过封装厂商命令行工具与系统命令来统一输出磁盘信息，主要覆盖：

- RAID 控制器识别
- 物理盘信息采集
- 故障盘定位灯开关
- 新盘制作 RAID0
- 故障盘信息上报
- 通过轻量 Web 页面展示磁盘状态

当前代码中实际支持的 RAID 工具/设备路径包括：

- `MegaCli`
- `StorCLI`
- `Arcconf`
- `NVMe` 直连盘采集

`docs/sas3ircu_cmd_result.txt` 目前仅作为历史样例保留，仓库内暂无对应适配器实现。

# 概念

对 RAID 阵列卡中的硬盘操作时，会有额外的信息需要提供，例如 `C:E:S` 与 `VD`。

- `C`：`Controller ID`
- `E`：`Enclosure Device ID`
- `S`：`Slot Number`
- `VD`：逻辑盘 ID，通常与 `Virtual Drive` 接近，但并不总是完全一致

# 常用命令

```bash
# 查看控制器与工具探测结果
gstor check

# 查看磁盘列表
gstor list

# 以 JSON 输出磁盘列表
gstor list -f json

# 打开 / 关闭定位灯
gstor locate on 0:1:2
gstor locate off 0:1:2

# 生成本地配置文件
gstor init

# 查看关键 SMART 信息（支持盘符或槽位）
gstor smart sda
gstor smart 0:24:15
gstor smart -v 0:24:15
gstor smart -f json nvme0n1

# 启动轻量 Web 页面
gstor server -p 9100
```

如需调试命令执行过程，可使用全局参数：

```bash
gstor -d list
gstor --debug locate on 0:1:2
```

更多说明见 `docs/DEBUG_MODE.md`。

# 本地开发

```bash
# 执行与 CI 接近的一组本地检查
./check.sh

# 构建本地二进制
./build.sh

# 或直接使用 Go 工具链
go test ./...
go build ./...
```

CI 当前会执行：

- `go mod download && go mod verify`
- `go vet ./...`
- `gofmt -s -l .`
- `go test -v ./...`
- `go build -v ./...`
- `golangci-lint`

# 项目结构

- `main.go`：程序入口
- `cmd/`：CLI 命令与 HTTP 入口层
- `common/controller/`：控制器探测与工具选择
- `common/block/`：各 RAID/NVMe 设备信息采集与定位逻辑
- `common/utils/`：shell 执行、debug 日志等通用能力
- `docs/`：专题说明文档

# 代码与开发规范

这部分用于约束后续改动，优先保证可维护性、可诊断性和现场安全性。

## 1. 分层边界

- `cmd/` 只负责参数解析、输出格式化、HTTP/CLI 边界处理
- `common/block/` 与 `common/controller/` 负责领域逻辑，不应直接 `os.Exit`
- 可复用逻辑优先下沉到 `common/`，避免在多个命令里复制

## 2. Shell 命令规范

- 新代码必须优先使用 `common/utils/shell.go` 中的 `ExecShell()` 或 `ExecShellWithShell()`
- `common/block/block.go` 中的 `Bash()` 仅用于兼容旧逻辑，不应继续扩散
- 能不用 shell 拼接就不要拼接；涉及用户输入时，优先做严格校验或改为参数化调用
- 需要 Bash 特性的场景，显式使用 `ExecShellWithShell(..., "/bin/bash")`

详细迁移规范见 `docs/SHELL_COMMAND_MIGRATION.md`。

## 3. 异常处理规范

- 库函数优先返回 `error`，不要在库层直接 `panic`、`os.Exit` 或吞错
- CLI 边界统一使用 `cobra.CheckErr(...)`
- HTTP 边界统一使用 `http.Error(...)`
- 可降级的错误要明确标记为 warning，并尽量输出到 `stderr` 或 debug 日志，而不是污染标准输出
- 非关键命令允许降级时，要保留足够上下文，便于定位问题

## 4. 输入校验规范

- 对 `C:E:S`、`C:S` 等磁盘定位参数做统一解析，不要到处直接 `strings.Split(...)[i]`
- 外部输入必须先校验再参与命令构造
- 需要兼容多种控制器输出格式时，先抽象 parser，再落到各适配器中

## 5. 重复代码治理

- Vendor 归一化、IP 探测、槽位解析、容量格式化等公共逻辑应尽量抽到公共函数
- 同一段 shell 管道出现两次及以上时，应评估是否值得抽象
- 如果复制是暂时的，必须在 PR/提交说明里注明后续收敛计划

## 6. 输出与兼容性

- 面向脚本消费的输出必须稳定，尤其是 `list -f` 这类 JSON 输出
- 面向人类的提示信息与面向机器的结构化输出不要混用在同一个 stdout 流里
- 如果调整字段名、字段类型或排序，需要同时更新文档并评估兼容性影响

## 7. 测试规范

- 修改 `controller` 选择逻辑时，至少补充映射与回退路径测试
- 修改 `block` 解析逻辑时，优先补 fixture / sample output 测试
- 没有真实硬件时，尽量用样例输出覆盖文本解析分支
- 提交前至少执行一次 `./check.sh`

## 8. 文档维护规范

- 新增命令、变更输出、修改调试方式时，同步更新 `README.md` 与相关 `docs/`
- 项目级中长期计划统一维护在 `ROADMAP.md`
- 若 README 与实现不一致，应优先修正文档或在变更中一并修复

# 相关文档

- 调试说明：`docs/DEBUG_MODE.md`
- Shell 命令迁移规范：`docs/SHELL_COMMAND_MIGRATION.md`
- 架构说明：`ARCHITECTURE.md`
- 路线图：`ROADMAP.md`
