# Docker 开发环境记录

## Docker 是什么

Docker 可以理解为一种“把程序运行环境打包起来”的工具。

平时直接在本机运行项目时，需要本机安装：

```text
Go
Node.js
npm
SQLite 相关编译依赖
```

如果换一台电脑，或者同学的系统版本不同，就可能出现：

```text
我的机器能跑，你的机器不能跑。
```

Docker 的思路是：不要假设每台电脑都已经装好了正确环境，而是把运行环境写成配置，让 Docker 根据配置创建一个隔离的运行空间。

在这个项目里，我们希望 Docker 帮我们准备：

```text
一个能运行 Go 后端的环境
一个能运行 Vue/Vite 前端的环境
```

这样你本机只需要安装 Docker，不需要手动安装完全相同版本的 Go 和 Node。

## 几个核心概念

### 镜像 image

镜像可以理解为“环境模板”。

例如：

```text
golang:1.26.2-bookworm
node:22-bookworm
```

它们分别表示：

```text
带 Go 1.26.2 的 Linux 环境
带 Node.js 22 的 Linux 环境
```

镜像本身不是正在运行的程序，只是一个可以用来创建运行环境的模板。

### 容器 container

容器是“由镜像启动出来的运行实例”。

类比一下：

```text
镜像 = 类 / 模板
容器 = 根据模板创建出来的对象 / 正在运行的实例
```

本项目启动后会有两个主要容器：

```text
backend 容器
  在里面执行 go run ./cmd/server

frontend 容器
  在里面执行 npm run dev
```

### 端口映射 ports

容器内部有自己的网络空间。后端在容器里监听 `8080`，不代表你本机浏览器一定能直接访问。

所以需要端口映射：

```yaml
ports:
  - "8080:8080"
```

含义是：

```text
宿主机 8080 端口 → 容器内部 8080 端口
```

前端也是一样：

```yaml
ports:
  - "5173:5173"
```

所以你可以在本机浏览器访问：

```text
http://localhost:5173
```

### volume

容器默认是相对临时的。容器删掉后，容器内部写入的东西也可能跟着消失。

volume 用来解决两个问题：

1. 把本机代码目录挂进容器，让容器看到你的源码。
2. 缓存依赖和构建结果，避免每次启动都重新下载。

例如：

```yaml
volumes:
  - .:/app
```

含义是：

```text
把当前项目目录挂载到容器里的 /app。
```

所以容器内执行：

```bash
go run ./cmd/server
```

实际用的是你本机这份仓库代码。

### Docker Compose

Docker 本身可以启动单个容器。

但本项目不是一个进程，而是两个进程：

```text
后端 Go server
前端 Vite dev server
```

它们还要互相通信：

```text
frontend 请求 /api → 转发给 backend
```

Docker Compose 就是用一个 `docker-compose.yml` 文件描述“这一组容器应该怎么一起启动、怎么连起来”。

你可以把 Compose 理解为：

```text
多容器项目的启动说明书。
```

## 2026-05-13：用 Docker Compose 建立前后端开发环境

本项目现在可以用 Docker Compose 启动一套本地开发环境：

```text
backend   Go 后端，监听 8080
frontend  Vue + Vite 前端，监听 5173
```

## 为什么用 Docker Compose

单独用 Docker 跑一个容器也可以，但本项目有两个开发进程：

```text
go run ./cmd/server
npm run dev
```

它们需要同时运行，而且前端 Vite 需要把 `/api` 请求转发给后端。Docker Compose 适合描述这种多服务开发环境。

## 新增文件

```text
docker-compose.yml
```

核心内容：

```yaml
services:
  backend:
    image: golang:1.26.2-bookworm
    working_dir: /app
    command: sh -c "go run ./cmd/server"
    ports:
      - "8080:8080"
    volumes:
      - .:/app

  frontend:
    image: node:22-bookworm
    working_dir: /app
    command: sh -c "npm install && npm run dev -- --host 0.0.0.0"
    ports:
      - "5173:5173"
    environment:
      VITE_API_PROXY_TARGET: http://backend:8080
    volumes:
      - ./web:/app
    depends_on:
      - backend
```

实际文件里还增加了几个 Docker volume：

```text
go-build-cache
go-mod-cache
web-node-modules
```

它们用于缓存 Go 编译结果、Go module 和前端依赖，避免每次启动都重新下载和编译。

## Vite 代理为什么要改

原来 `web/vite.config.js` 里写的是：

```js
'/api': 'http://localhost:8080'
```

这在本机直接运行时没问题，但在 Docker 容器里不对。

原因：

```text
frontend 容器里的 localhost 指的是 frontend 容器自己，
不是 backend 容器，也不是宿主机。
```

所以 Docker Compose 中应该让前端访问服务名：

```text
http://backend:8080
```

现在 `vite.config.js` 改成：

```js
server: {
  host: '0.0.0.0',
  proxy: {
    '/api': process.env.VITE_API_PROXY_TARGET || 'http://localhost:8080',
  },
}
```

含义：

```text
本机直接 npm run dev：
  没有 VITE_API_PROXY_TARGET，默认代理到 http://localhost:8080。

Docker Compose：
  设置 VITE_API_PROXY_TARGET=http://backend:8080，
  前端容器通过 Compose 服务名访问后端容器。
```

## 启动方式

在项目根目录运行：

```bash
docker compose up
```

第一次运行会下载镜像并安装依赖，时间会比较久。

启动后访问：

```text
前端页面：http://localhost:5173
后端接口：http://localhost:8080/api/health
```

如果想后台运行：

```bash
docker compose up -d
```

查看日志：

```bash
docker compose logs -f
```

只看后端日志：

```bash
docker compose logs -f backend
```

只看前端日志：

```bash
docker compose logs -f frontend
```

## 停止方式

停止容器：

```bash
docker compose down
```

停止并删除依赖缓存 volume：

```bash
docker compose down -v
```

一般开发时不要随便加 `-v`，否则下次会重新安装依赖。

## 在容器里运行命令

运行 Go 测试：

```bash
docker compose run --rm backend go test ./...
```

运行前端构建：

```bash
docker compose run --rm frontend npm run build
```

进入后端容器 shell：

```bash
docker compose run --rm backend bash
```

进入前端容器 shell：

```bash
docker compose run --rm frontend bash
```

## 文件同步

Compose 中配置了 bind mount：

```yaml
volumes:
  - .:/app
  - ./web:/app
```

所以本机修改代码后，容器里能立即看到。

注意：

```text
Vite 前端通常会自动热更新。
Go 后端当前只是 go run，不带热重载。
```

如果改了 Go 代码，需要重启后端容器：

```bash
docker compose restart backend
```

## 数据库文件

后端会在 `data/` 下生成 SQLite 数据库文件，例如：

```text
data/requests_5e_20f_scan_*.db
```

因为项目根目录挂载进了后端容器，所以这些数据库文件会保存在本机仓库目录中。

`.gitignore` 已经忽略：

```text
data/*.db
data/*.db-shm
data/*.db-wal
```

所以数据库运行产物不会被提交。

## 常见问题

### 1. 端口被占用

如果本机已经有服务占用 8080 或 5173，Compose 会启动失败。

解决方式：

```bash
docker compose down
```

或者修改 `docker-compose.yml` 的端口映射，例如：

```yaml
ports:
  - "5174:5173"
```

表示宿主机访问 5174，容器内部仍然是 5173。

### 2. 前端能打开但 API 请求失败

检查：

```bash
docker compose logs -f backend
docker compose logs -f frontend
```

重点看 `frontend` 是否带了：

```text
VITE_API_PROXY_TARGET=http://backend:8080
```

以及后端是否正常监听：

```text
server listening on http://localhost:8080
```

### 3. Go 依赖下载很慢

第一次启动需要下载 Go module 和 npm 包。

后续会复用：

```text
go-mod-cache
go-build-cache
web-node-modules
```

如果执行了：

```bash
docker compose down -v
```

缓存会被删掉，下次会重新下载。

## 当前建议工作流

开发时：

```bash
docker compose up
```

浏览器访问：

```text
http://localhost:5173
```

修改前端代码后，Vite 自动刷新。

修改后端 Go 代码后：

```bash
docker compose restart backend
```

提交前验证：

```bash
docker compose run --rm backend go test ./...
docker compose run --rm frontend npm run build
```
