![gstor](https://user-images.githubusercontent.com/21356580/216275709-3ed3420b-5c35-47d7-b824-582bcf933c4d.jpeg)
Gstor 是一个应用于 Linux 服务器上的硬盘管理工具。

[![](https://github.com/serialq7ic4/gstor/workflows/test/badge.svg)](https://github.com/serialq7ic4/gstor/workflows/test/badge.svg)

# 概述

Gstor 是一个使用 Cobra 构建的命令行工具，它依赖 Megacli、Storcli、Sas3ircu 和 Arcconf 等命令行工具，实现了常用硬盘管理的功能集成。

目前包含的功能如下：
- 查看控制器
- 查看硬盘
- 定位硬盘
- 给新盘做 raid0

# 概念

对 RAID 阵列卡中的硬盘操作时，会有额外的信息需要提供，例如 C:E:S 的值, 以及 VD 的值，这些称呼在不同的命令行工具中都是共通的。

C 指的是 Controller ID

E 指的是 Enclosive Device ID

S 指的是 Slot Number

VD 指的是做了 raid 的硬盘的逻辑盘的 ID，即 Target ID，一般与 Virtual Drive 一致，但也有不一致的情况