# 电梯调度算法可视化程序

## 项目简介

实现一个电梯调度系统，支持多种调度算法（FCFS、SCAN、LOOK、First Available、Nearest Idle），并提供 Web 前端可视化展示电梯运行状态。

## 技术栈

- 后端：Go（标准库 `net/http`）
- 数据库：SQLite
- 前端：Vue 3 + Element Plus + Vite

## 快速开始

```bash
# 启动后端
go run ./cmd/server

# 前端开发（可选，web/dist 已包含构建产物）
cd web && npm install && npm run dev
```

浏览器访问 `http://localhost:8080`。

## 项目结构

```text
cmd/server/             程序入口
internal/elevator/      电梯核心逻辑（模型、调度接口、多种调度算法、SQLite 请求持久化）
internal/api/           HTTP API handler（路由注册、自动步进）
web/                    前端页面（Vue 3 + Element Plus）
web/dist/               前端构建产物
docs/                   学习记录
```
