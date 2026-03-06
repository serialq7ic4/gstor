# 架构说明

本文档描述 `gstor` 当前的主要执行链路、代码分层和关键约束，帮助后续重构时保持一致的设计方向。

## 1. 总体分层

项目目前分为四层：

- `main.go`
  - 程序入口，只负责调用 `cmd.Execute()`
- `cmd/`
  - CLI 命令、HTTP 边界、输出格式化
  - 不承载复杂解析逻辑
- `common/controller/`
  - 控制器探测、控制器型号到工具路径的选择
- `common/block/`
  - RAID / NVMe 设备采集、定位灯操作、磁盘信息聚合
- `common/utils/`
  - shell 执行、debug、网络/IP 探测等公共能力

## 2. 主要执行链路

### 2.1 设备采集

`gstor list` / `gstor server` 的核心链路如下：

1. `cmd` 层调用 `block.Devices()`
2. `block.Devices()` 优先读取配置中的 `controller.tool`
3. 若配置未指定，则调用 `controller.Collect()` 自动探测
4. 根据探测结果选择对应 collector：
   - `megacliCollector`
   - `storcliCollector`
   - `arcconfCollector`
   - `nvmeCollector`
5. 若存在 RAID collector，则通过 `combinedCollector` 合并 RAID 与 NVMe 结果

### 2.2 定位灯操作

`gstor locate on/off` 与 Web 定位接口共用 `DiskCollector` 接口：

- `TurnOn(slot string) error`
- `TurnOff(slot string) error`

当前约定：

- RAID 盘位参数使用统一解析器，支持 `c:s` 与 `c:e:s`
- 成功返回 `nil`
- 失败返回 `error`
- NVMe 当前不支持定位灯操作

### 2.3 故障上报

`gstor report` 会：

1. 采集设备信息
2. 过滤介质错误或异常状态硬盘
3. 获取主 IP 与机器序列号
4. 组装 JSON 后发送给外部 API

## 3. 关键设计点

### 3.1 适配器模式

`common/block/` 通过适配器模式隔离不同厂商工具的差异：

- 每个工具实现自己的 collector
- `Devices()` 根据控制器选择具体实现
- 调用方只依赖统一的 `DiskCollector` 接口

### 3.2 Shell 依赖

当前系统仍高度依赖 shell 命令和厂商 CLI：

- `lspci`
- `lsblk`
- `smartctl`
- `MegaCli`
- `StorCLI`
- `Arcconf`

因此后续修改时要特别关注：

- 命令超时
- 输出格式漂移
- 错误上下文
- stdout/stderr 边界

### 3.3 输出约束

项目同时面向两类消费者：

- 人工使用者：表格、提示信息、Web 页面
- 机器使用者：JSON 输出、脚本集成、上报接口

后续改动应避免将 warning 或调试信息混入机器可读输出。

## 4. 当前演进方向

结合 `ROADMAP.md`，当前架构上的主要演进方向是：

- 持续减少直接 `strings.Split(...)[i]` 和 shell 拼接
- 逐步迁移 `Bash()` 老路径，保留明确错误语义
- 补齐解析逻辑测试，降低厂商输出变更带来的风险
- 继续收敛 CLI、HTTP、collector 三层边界
