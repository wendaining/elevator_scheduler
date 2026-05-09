# 电梯调度算法可视化程序

同济大学 软件工程专业 操作系统课程 项目一

> [!CAUTION]
> 未完成

## 项目简介

实现一个电梯调度系统，支持多种调度算法（先来先服务、最短寻道优先、SCAN 等），并提供 Web 前端可视化展示电梯运行状态。

## 技术栈

- 后端：Go（标准库 `net/http`）
- 数据库：SQLite
- 前端：原生 HTML / CSS / JavaScript

## 快速开始

```bash
go run ./cmd/server
```

浏览器访问 `http://localhost:8080`。

## 项目结构

```text
cmd/server/        程序入口
internal/elevator/ 电梯系统核心逻辑（模型、调度算法）
internal/api/      HTTP API handler
web/               前端页面
docs/              学习记录
```
