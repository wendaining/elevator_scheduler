这是用于记录项目开发历程的文档，主要是由我自己来编写，当然也可以让 Agent 为我代为编写，但是目的其实主要是记录自己的学习和完成进度

目录已建立、Go module 已初始化、下一步准备做核心模型

## 2026-05-06：建立最小 Go 后端

本阶段目标：先不实现电梯调度算法，只让 Go 后端程序能够启动，并提供一个最小 HTTP API。这样可以先理解“浏览器 / curl 如何访问 Go 程序”，也就是前后端通信的最小基础。

### 本次新增文件

- `cmd/server/main.go`：Go 后端程序入口。

### 如何运行

在项目根目录执行：

```bash
go run ./cmd/server
```

如果程序正常启动，终端会输出类似：

```text
server listening on http://localhost:8080
```

此时可以在浏览器访问：

```text
http://localhost:8080/
```

也可以访问健康检查接口：

```text
http://localhost:8080/api/health
```

或者使用命令：

```bash
curl http://localhost:8080/api/health
```

期望返回：

```json
{"status":"ok"}
```

### 代码阅读顺序

建议先看 `cmd/server/main.go`，按下面顺序理解：

1. `package main`

   Go 程序的可执行入口包必须叫 `main`。这个文件最终会被 `go run ./cmd/server` 编译并运行。

2. `import (...)`

   当前只用了 Go 标准库：

   - `encoding/json`：把 Go 里的数据编码成 JSON。
   - `log`：在终端打印日志，并在严重错误时退出程序。
   - `net/http`：Go 标准库里的 HTTP 服务器功能。

3. `func main()`

   `main` 函数是程序启动后最先执行的函数。本项目里的 `main` 暂时只做三件事：

   - 创建路由表 `mux`
   - 注册 URL 和处理函数
   - 启动 HTTP 服务

4. `http.NewServeMux()`

   `ServeMux` 可以理解为“路由表”。它负责决定不同 URL 应该交给哪个函数处理。

   当前注册了两个路由：

   - `/`：交给 `handleHome`
   - `/api/health`：交给 `handleHealth`

5. `http.ListenAndServe(":8080", mux)`

   这一行让程序监听本机的 `8080` 端口。只要程序不退出，它就会一直等待浏览器或其他客户端发来的 HTTP 请求。

6. `handleHome`

   这是访问首页 `/` 时执行的函数。它返回一段普通文本：

   ```text
   Elevator scheduler server is running.
   ```

   这个接口不是核心功能，只是方便确认服务器已经启动。

7. `handleHealth`

   这是访问 `/api/health` 时执行的函数。它返回 JSON：

   ```json
   {"status":"ok"}
   ```

   健康检查接口常用于确认后端服务是否还活着。它暂时不涉及电梯业务逻辑。

### 两个 handler 函数的实现细节

在 Go 的 `net/http` 里，一个 HTTP 处理函数通常长这样：

```go
func handler(w http.ResponseWriter, r *http.Request)
```

这里有两个重要参数：

- `w http.ResponseWriter`：用来写 HTTP 响应。比如设置响应头、写入文本、写入 JSON、返回错误状态码。
- `r *http.Request`：表示客户端发来的 HTTP 请求。比如请求方法、请求路径、请求头、请求体都可以从这里读取。

也就是说，handler 函数的角色是：

```text
读取请求 r
    ↓
执行一些判断或业务逻辑
    ↓
通过 w 写回响应
```

#### `handleHome`

当前实现：

```go
func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Elevator scheduler server is running.\n"))
}
```

这个函数处理首页 `/`。

第一段：

```go
if r.URL.Path != "/" {
	http.NotFound(w, r)
	return
}
```

这是为了避免所有未知路径都被首页处理掉。因为 Go 的 `ServeMux` 对 `/` 的匹配比较特殊，`/` 会匹配很多路径，例如 `/abc` 也可能进入 `handleHome`。所以这里手动判断：只有路径严格等于 `/` 时才返回首页文本，否则返回 `404 Not Found`。

第二段：

```go
w.Header().Set("Content-Type", "text/plain; charset=utf-8")
```

这是设置响应类型，告诉浏览器：这次返回的是普通文本，编码是 UTF-8。

第三段：

```go
_, _ = w.Write([]byte("Elevator scheduler server is running.\n"))
```

这是把响应内容写回客户端。`w.Write` 需要 `[]byte`，所以要把字符串转换成字节切片。前面的 `_, _ =` 表示暂时忽略返回的字节数和错误。这里是一个很小的健康提示页面，所以暂时这样写可以接受。

#### `handleHealth`

当前实现：

```go
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]string{
		"status": "ok",
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
```

这个函数处理 `GET /api/health`。

第一段：

```go
if r.Method != http.MethodGet {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return
}
```

这是限制这个接口只能用 `GET` 请求访问。如果客户端用 `POST`、`PUT` 等方法访问，就返回 `405 Method Not Allowed`。这就是 API 设计里常见的“同一个路径，不同 HTTP 方法表示不同操作”的基础。

#### HTTP 方法：GET、POST、PUT 是什么

HTTP 请求不只有 URL，还有一个“方法”（method）。方法用来表达这次请求大概想做什么。

可以先这样理解：

```text
GET   读取数据
POST  新增数据或提交一个动作
PUT   整体更新一个已有资源
```

更具体一点：

- `GET`：向后端“要数据”。例如查看当前电梯状态、查看健康检查结果。`GET` 请求通常不应该改变后端状态。
- `POST`：向后端“提交数据”或“触发动作”。例如用户按了某层的上行按钮，前端可以用 `POST /api/request` 告诉后端新增一个乘梯请求。
- `PUT`：向后端“更新某个已有对象”。例如如果以后有接口用于修改系统配置，比如把楼层数从 20 改成 10，可能会设计成 `PUT /api/config`。

在当前项目里，后续可能会有这样的 API：

```text
GET  /api/state    获取电梯系统当前状态
POST /api/request  提交一个新的楼层请求
PUT  /api/config   更新系统配置，例如楼层数、电梯数量
```

所以这段代码：

```go
if r.Method != http.MethodGet {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return
}
```

意思是：`/api/health` 这个接口只是用来读取服务是否正常，不应该修改任何状态，所以它只接受 `GET`。如果有人用 `POST /api/health` 或 `PUT /api/health` 访问，后端就会拒绝，并返回 `405 Method Not Allowed`。

用命令可以这样感受区别：

```bash
curl http://localhost:8080/api/health
```

这默认是 `GET` 请求，所以会返回：

```json
{"status":"ok"}
```

如果手动指定 `POST`：

```bash
curl -X POST http://localhost:8080/api/health
```

后端会返回错误，因为当前代码明确禁止对 `/api/health` 使用 `POST`。

第二段：

```go
response := map[string]string{
	"status": "ok",
}
```

这里创建了一个简单的 Go map。它后面会被编码成 JSON：

```json
{"status":"ok"}
```

第三段：

```go
w.Header().Set("Content-Type", "application/json; charset=utf-8")
```

这是告诉客户端：这次响应是 JSON。后续前端 JavaScript 用 `fetch` 请求接口时，就可以把响应当作 JSON 数据解析。

第四段：

```go
if err := json.NewEncoder(w).Encode(response); err != nil {
	http.Error(w, "failed to encode response", http.StatusInternalServerError)
}
```

`json.NewEncoder(w).Encode(response)` 的意思是：把 `response` 编码成 JSON，并直接写入 HTTP 响应。这里如果编码失败，就返回 `500 Internal Server Error`。

对于当前这个简单 map，编码基本不会失败。但保留错误处理是一个好习惯，因为以后返回结构体、复杂数据时，错误处理会更重要。

### 什么是 API

API 可以先粗略理解为“前端和后端约定好的通信入口”。

比如当前这个接口：

```text
GET /api/health
```

它约定了几件事：

- 请求路径是 `/api/health`
- 请求方法是 `GET`
- 返回格式是 JSON
- 正常返回内容是 `{"status":"ok"}`

之后电梯项目会继续增加这样的 API，例如：

```text
GET  /api/state
POST /api/request
```

前端不需要知道 Go 后端内部有多少结构体、多少调度算法。前端只需要按照 API 约定发送请求，并读取 JSON 响应。

因此前后端通信的关键不是“前端直接调用 Go 函数”，而是：

```text
前端 JavaScript
    ↓ HTTP 请求
后端 API handler
    ↓ 调用 Go 内部业务逻辑
电梯系统核心代码
```

### handler 函数应该放在哪里

当前 `handleHome` 和 `handleHealth` 放在 `cmd/server/main.go` 里，是为了第一步学习时足够直观：打开一个文件就能看到程序如何启动、如何注册路由、请求来了以后如何返回响应。

但是从项目结构上说，随着功能增加，把 handler 全部放在 `main.go` 里并不合适。原因是：

- `main.go` 会越来越长。
- 程序启动逻辑和 API 处理逻辑会混在一起。
- 后续 `GET /api/state`、`POST /api/request` 会需要调用电梯系统，如果都写在 `main.go` 里会不清晰。

所以更合理的长期结构是：

```text
cmd/server/main.go       负责启动服务、创建系统、注册路由
internal/api/handler.go  负责 HTTP API 的具体处理函数
internal/elevator/       负责电梯系统核心逻辑
```

也就是说，现在这样写是一个教学阶段的最小实现；等开始实现 `GET /api/state` 和 `POST /api/request` 时，就应该把 API handler 移到 `internal/api/` 中。

可以把演进路线理解为：

```text
第一步：handler 暂时放 main.go，方便理解 HTTP 服务最小结构
第二步：出现业务 API 后，把 handler 移到 internal/api/
第三步：main.go 只负责组装，不负责具体业务处理
```

### 这一步和前后端通信有什么关系

前端和后端通信，本质上就是：

```text
浏览器 / JavaScript 发送 HTTP 请求
        ↓
Go 后端根据 URL 找到处理函数
        ↓
处理函数返回文本或 JSON
        ↓
浏览器 / JavaScript 使用返回结果更新页面
```

当前我们还没有写前端页面，但 `/api/health` 已经是一个真正的后端 API。后续前端 JavaScript 可以通过 `fetch("/api/health")` 或 `fetch("/api/state")` 请求后端数据。

### 浏览器 / JavaScript 如何发送 HTTP 请求

浏览器发送 HTTP 请求有几种常见方式。

#### 1. 在地址栏输入 URL

例如在浏览器地址栏输入：

```text
http://localhost:8080/api/health
```

浏览器会自动向后端发送一个 `GET` 请求：

```text
GET /api/health
```

这和执行下面的命令很像：

```bash
curl http://localhost:8080/api/health
```

所以，当我们只是想“查看一个接口返回了什么”，可以直接用浏览器地址栏访问。

但地址栏访问通常只能方便地发送 `GET` 请求。如果要发送 `POST`，就需要表单、JavaScript 或专门的 API 测试工具。

#### 2. HTML 表单提交请求

传统网页可以用 `<form>` 提交请求，例如：

```html
<form method="post" action="/api/requests">
  <input name="floor" value="5">
  <button type="submit">submit</button>
</form>
```

点击按钮后，浏览器会向 `/api/requests` 发送一个 `POST` 请求。

不过这个项目后续会更倾向于用 JavaScript 控制页面，因为我们希望点击按钮后局部刷新电梯状态，而不是每次都重新加载整个页面。

#### 3. JavaScript 使用 `fetch`

现代前端最常用的方式是 `fetch`。它可以在不刷新整个页面的情况下，向后端发送 HTTP 请求。

读取健康检查接口：

```js
const response = await fetch("/api/health");
const data = await response.json();
console.log(data);
```

这段代码会发送：

```text
GET /api/health
```

然后把后端返回的 JSON 解析成 JavaScript 对象。对于当前接口，`data` 大概是：

```js
{
  status: "ok"
}
```

后续读取电梯系统状态时，可能会写成：

```js
const response = await fetch("/api/state");
const state = await response.json();
```

提交一个楼层请求时，可能会写成：

```js
await fetch("/api/requests", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    floor: 5,
    direction: "up",
  }),
});
```

这段代码会发送一个 `POST` 请求，并把 JavaScript 对象转换成 JSON 请求体：

```json
{
  "floor": 5,
  "direction": "up"
}
```

后端的 handler 会从请求体里读出这些数据，然后把它转换成 Go 里的结构体或变量，再交给电梯系统核心逻辑处理。

#### 前后端通信的完整链路

以“用户点击 5 楼上行按钮”为例，后续页面里的过程大概是：

```text
用户点击按钮
    ↓
JavaScript 触发 click 事件
    ↓
fetch("/api/requests", { method: "POST", body: ... })
    ↓
Go 后端的 handler 收到 HTTP 请求
    ↓
handler 解析 JSON：floor = 5, direction = "up"
    ↓
handler 调用 elevator system 添加请求
    ↓
后端返回成功响应
    ↓
JavaScript 再 fetch("/api/state")
    ↓
页面根据最新状态重新显示电梯位置
```

所以，前端并不是直接操作 Go 代码里的变量。前端只能通过 HTTP API 和后端沟通。后端负责维护真正的系统状态，前端负责把状态展示出来，并把用户操作转换成 HTTP 请求。

### 当前还没有做的事

- 还没有电梯数据结构。
- 还没有调度算法。
- 还没有前端页面。
- 还没有前后端业务数据交互。

这一步只解决一个问题：Go 后端如何启动，并如何对 HTTP 请求返回 JSON。

### 本次验证结果

已执行代码格式化：

```bash
gofmt -w cmd/server/main.go
```

已执行基础检查：

```bash
go test ./...
```

结果：

```text
?   	os_sp26_proj1/cmd/server	[no test files]
```

这表示当前包可以正常编译，只是还没有测试文件。

已启动服务：

```bash
go run ./cmd/server
```

已验证首页：

```bash
curl http://localhost:8080/
```

返回：

```text
Elevator scheduler server is running.
```

已验证健康检查接口：

```bash
curl http://localhost:8080/api/health
```

返回：

```json
{"status":"ok"}
```

### 一个环境相关的小问题

在当前 Agent 的沙箱环境里，第一次运行 `go test ./...` 时，因为 Go 需要写入 `~/.cache/go-build`，遇到了只读文件系统问题；第一次运行 `go run ./cmd/server` 时，绑定 `localhost:8080` 也被沙箱限制。授权后重新运行通过。

如果我自己在正常 WSL 终端中运行，一般直接执行下面两个命令即可：

```bash
go test ./...
go run ./cmd/server
```
