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

## 2026-05-06：建立电梯核心数据模型骨架

本阶段目标：开始写 `internal/elevator/model.go`，先描述“电梯系统有哪些状态”，不急着写调度算法和并发逻辑。

这里采用“填空题式”的方式：Agent 先把最小可编译框架搭好，并在代码里用 `TODO(student)` 留出几个适合我自己补充或思考的位置。

### 本次新增文件

- `internal/elevator/model.go`：电梯系统核心数据模型。

### 为什么先写 model

在写调度算法之前，需要先明确系统里有哪些对象。

当前最核心的对象是：

```text
Direction   电梯方向
Request     一次乘梯请求
Elevator    一部电梯的状态
System      整个电梯系统的状态
```

后续的调度算法、HTTP API、前端页面，都会围绕这些数据结构展开。

### 代码阅读顺序

建议按这个顺序读 `internal/elevator/model.go`：

1. `package elevator`

   这表示当前文件属于 `elevator` 包。以后其他后端代码可以通过下面的方式使用它：

   ```go
   import "os_sp26_proj1/internal/elevator"
   ```

2. `type Direction string`

   这表示定义一个新类型 `Direction`，它底层是字符串。

   好处是代码会更清楚。例如：

   ```go
   DirectionUp
   DirectionDown
   DirectionIdle
   ```

   比到处写普通字符串 `"up"`、`"down"`、`"idle"` 更不容易写错。

3. `const (...)`

   `const` 用来定义常量，也就是运行时不会改变的值。

   当前方向常量是：

   ```go
   const (
   	DirectionIdle Direction = "idle"
   	DirectionUp   Direction = "up"
   	DirectionDown Direction = "down"
   )
   ```

4. `type RequestKind string`

   这个类型用来区分请求来源：

   - `RequestKindHall`：楼层外部按钮，例如 5 楼按上行。
   - `RequestKindCabin`：电梯内部按钮，例如进入电梯后按 12 楼。

5. `type Request struct`

   `Request` 表示一次乘梯请求。

   当前字段：

   ```go
   type Request struct {
   	Floor     int         `json:"floor"`
   	Direction Direction   `json:"direction"`
   	Kind      RequestKind `json:"kind"`
   }
   ```

   字段含义：

   - `Floor`：请求发生在哪一层，或者想去哪个楼层。
   - `Direction`：请求方向，例如上行或下行。
   - `Kind`：请求来源，是楼层外部请求还是电梯内部请求。

6. `type Elevator struct`

   `Elevator` 表示一部电梯当前的状态。

   当前字段：

   ```go
   type Elevator struct {
   	ID           int       `json:"id"`
   	CurrentFloor int       `json:"currentFloor"`
   	Direction    Direction `json:"direction"`
   	DoorOpen     bool      `json:"doorOpen"`
   	TargetFloors []int     `json:"targetFloors"`
   }
   ```

   字段含义：

   - `ID`：电梯编号。
   - `CurrentFloor`：当前楼层。
   - `Direction`：当前运行方向。
   - `DoorOpen`：门是否打开。
   - `TargetFloors`：当前目标楼层列表。

7. `type System struct`

   `System` 表示整个电梯调度系统。

   当前字段：

   ```go
   type System struct {
   	FloorCount      int        `json:"floorCount"`
   	Elevators       []Elevator `json:"elevators"`
   	PendingRequests []Request  `json:"pendingRequests"`
   }
   ```

   字段含义：

   - `FloorCount`：大楼总楼层数。
   - `Elevators`：所有电梯。
   - `PendingRequests`：还没有处理完成的请求。

### Go 语法细节

#### 为什么字段名首字母大写

Go 里，名字首字母大写表示“导出”，也就是其他包可以访问。

例如：

```go
CurrentFloor int
```

后续 `internal/api` 包需要读取电梯状态并返回 JSON，所以这些字段先用大写。

如果写成：

```go
currentFloor int
```

这个字段只能在 `elevator` 包内部访问，其他包不能直接读。

#### 什么是 `[]int`

`[]int` 表示 int 切片，可以先理解成“可变长度数组”。

例如：

```go
TargetFloors []int
```

表示一部电梯可能有多个目标楼层，例如：

```go
[]int{3, 8, 12}
```

#### 什么是 struct tag

例如：

```go
CurrentFloor int `json:"currentFloor"`
```

反引号里的内容叫 struct tag。它告诉 JSON 编码器：把这个字段输出成 JSON 时，字段名叫 `currentFloor`。

Go 字段名是：

```go
CurrentFloor
```

JSON 字段名是：

```json
"currentFloor"
```

这样前端 JavaScript 读起来更自然。

### 留给我完成的填空

当前 `model.go` 里有几个 `TODO(student)`：

1. 思考 `Request` 是否需要记录创建时间。

   这和 FCFS 有关。FCFS 是先来先服务，如果要严格比较“谁先来”，就可能需要记录请求创建时间。

2. 给 `Elevator` 增加报警 / 紧急停止相关字段。

   课程要求里提到了报警按钮。可以考虑加一个 `bool` 字段，例如表示“这部电梯是否处于紧急停止状态”。

3. 给 `System` 增加当前调度算法名称。

   后续如果支持算法切换，可以在系统状态里记录当前算法，例如 `"nearest"`、`"fcfs"`、`"scan"`。

这些 TODO 现在不急着全部完成。建议先读懂已有字段，再尝试补第 2 个：给 `Elevator` 增加一个报警状态字段。

### 当前还没有做的事

- 还没有 `NewSystem` 初始化函数。
- 还没有添加请求的方法。
- 还没有 `Step()` 模拟运行。
- 还没有调度算法。
- 还没有并发逻辑。

这一步只解决一个问题：先把电梯系统的核心状态描述出来。

## 2026-05-06：实现 `NewSystem`

本阶段目标：在 `internal/elevator/system.go` 中实现系统初始化函数。

### 本次新增文件

- `internal/elevator/system.go`：放和 `System` 相关的行为函数。

### 当前实现

```go
func NewSystem(floors int, elevatorCount int) *System {
	if floors < 1 {
		floors = 1
	}
	if elevatorCount < 1 {
		elevatorCount = 1
	}

	elevators := make([]Elevator, elevatorCount)
	for i := range elevators {
		elevators[i] = Elevator{
			ID:            i + 1,
			CurrentFloor: 1,
			Direction:    DirectionIdle,
			DoorOpen:     false,
			TargetFloors: []int{},
			EmergencyStop: false,
		}
	}

	return &System{
		FloorCount:      floors,
		Elevators:       elevators,
		PendingRequests: []Request{},
	}
}
```

### 这个函数做了什么

`NewSystem` 是一个构造函数。Go 没有 class 构造函数语法，所以项目里通常会写一个 `NewXxx` 函数来创建对象。

这个函数当前做了几件事：

1. 接收楼层数 `floors` 和电梯数量 `elevatorCount`。
2. 如果参数小于 1，就返回错误，不创建系统。
3. 用 `make([]Elevator, elevatorCount)` 创建指定数量的电梯切片。
4. 用 `for i := range elevators` 给每部电梯填入初始状态。
5. 返回一个 `*System`，也就是指向 `System` 的指针。

### Go 语法细节

#### 为什么返回 `(*System, error)`

```go
func NewSystem(...) (*System, error)
```

这里返回了两个值：

- `*System`：创建成功时返回系统对象。
- `error`：创建失败时返回错误原因。

可以先这样理解：后续系统会不断变化，例如添加请求、电梯移动、开门关门。如果多个函数都要操作同一个系统对象，使用指针更自然。

Go 通常不使用“抛异常 / 捕获异常”的方式处理普通业务错误，而是让函数显式返回 `error`。调用方必须自己判断错误：

```go
system, err := elevator.NewSystem(20, 5)
if err != nil {
	log.Fatal(err)
}
```

这样做的好处是错误不会被偷偷吞掉。比如如果调用方传入：

```go
elevator.NewSystem(0, 5)
```

这明显是不合理的楼层数。与其强行把 `0` 改成 `1`，不如返回错误，让调用方知道自己传错了参数。

当前实现使用：

```go
return nil, fmt.Errorf("floors must be at least 1, got %d", floors)
```

含义是：

- 第一个返回值是 `nil`，表示没有成功创建 `System`。
- 第二个返回值是一个错误，说明失败原因。

#### 什么是 `make([]Elevator, elevatorCount)`

`make` 可以创建 slice、map、channel。

这里：

```go
elevators := make([]Elevator, elevatorCount)
```

表示创建一个长度为 `elevatorCount` 的 `[]Elevator`。

如果 `elevatorCount` 是 5，那么会得到 5 个电梯位置，后面循环会逐个填入初始值。


### 本次验证

已运行：

```bash
gofmt -w internal/elevator/system.go
go test ./...
```

结果通过：

```text
?   	os_sp26_proj1/cmd/server          [no test files]
?   	os_sp26_proj1/internal/elevator   [no test files]
```

## 2026-05-06：实现 `Snapshot`

本阶段目标：给 `System` 增加一个查看当前状态的方法，先返回 JSON，方便后续调试和 HTTP API 使用。

### 当前实现

```go
func (s *System) Snapshot() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}
```

### 这个函数做了什么

`Snapshot` 的意思是“快照”：在某一时刻，把系统当前状态拿出来看一眼。

当前系统状态包括：

- `FloorCount`
- `Elevators`
- `PendingRequests`

因为这些字段在 `model.go` 里都写了 JSON tag，所以 `Snapshot()` 可以直接把整个 `System` 编码成 JSON。

### 为什么返回 `[]byte, error`

```go
([]byte, error)
```

表示这个函数返回两个值：

- `[]byte`：JSON 数据本身。HTTP 响应、文件写入、终端打印都可以使用字节切片。
- `error`：JSON 编码失败时的错误。

调用方以后可以这样使用：

```go
data, err := system.Snapshot()
if err != nil {
	return err
}
fmt.Println(string(data))
```

这里的：

```go
string(data)
```

是把 `[]byte` 转换成字符串，方便打印出来看。

### 为什么用 `json.MarshalIndent`

`json.MarshalIndent` 会生成带缩进的 JSON，比普通 `json.Marshal` 更适合人阅读。

例如结果大概会像这样：

```json
{
  "floorCount": 20,
  "elevators": [
    {
      "id": 1,
      "currentFloor": 1,
      "direction": "idle",
      "doorOpen": false,
      "targetFloors": [],
      "emergencyStop": false
    }
  ],
  "pendingRequests": []
}
```

后续真正写 HTTP API 时，可以直接把这段 JSON 写入响应：

```go
w.Header().Set("Content-Type", "application/json; charset=utf-8")
w.Write(data)
```

不过到 API 阶段，也可以选择让 handler 自己调用 `json.NewEncoder(w).Encode(...)`。当前 `Snapshot()` 返回 JSON 主要是为了让模型阶段容易观察。

## 2026-05-06：实现同步版 `Step`

本阶段目标：先实现一个最简单的同步模拟步骤，让系统状态可以随着一次次调用 `Step()` 发生变化。

### 当前策略

这不是最终调度算法，只是一个临时的最小策略：

```text
如果有待处理请求，并且 1 号电梯空闲：
    把最早的请求分配给 1 号电梯

然后遍历所有电梯：
    如果电梯有目标楼层，就向目标楼层移动一层
    如果电梯已经到达目标楼层，就开门并移除这个目标
    如果电梯没有目标楼层，就保持 idle
```

也就是说，当前版本还没有真正比较“哪部电梯更合适”，只是为了让系统先能动起来。

### 当前实现涉及的函数

```go
func (s *System) Step() error
```

这是对外使用的模拟入口。每调用一次，系统推进一个离散时间片。

```go
func (s *System) assignNextRequestToFirstElevator()
```

这是临时调度策略：只把请求分配给第一部空闲电梯。

```go
func stepElevator(e *Elevator)
```

这是推进单部电梯的函数。每次最多让电梯移动一层，或者在到达目标楼层时开门。

### 为什么没有请求时不返回错误

之前可能会想到：如果没有 `PendingRequests`，就返回错误。

但更合理的设计是：没有请求时，系统只是空闲，调用 `Step()` 什么都不做即可。这不是程序错误。

所以当前 `Step()` 只有在系统本身不合法时才返回错误，例如：

```go
if len(s.Elevators) == 0 {
	return fmt.Errorf("system has no elevators")
}
```

### 一个例子

假设：

```text
1 号电梯初始在 1 楼
PendingRequests 里有一个请求：去 4 楼
```

连续调用：

```go
system.Step()
system.Step()
system.Step()
system.Step()
```

状态变化大致是：

```text
Step 1：请求分配给 1 号电梯，电梯从 1 楼到 2 楼
Step 2：电梯从 2 楼到 3 楼
Step 3：电梯从 3 楼到 4 楼
Step 4：电梯到达目标楼层，方向变 idle，开门，并移除目标楼层
```

### 当前限制

- 只会把请求分配给 1 号电梯。
- 只有当 1 号电梯没有目标楼层时，才会分配下一个请求。
- 暂时没有最近电梯、FCFS、SCAN 等真正调度算法。
- 暂时没有 goroutine 和 channel。
- `DoorOpen` 会在下一次 `Step()` 里自动关闭。

这些限制是有意保留的。当前目标是让同步模型可运行、可观察，后续再替换成更合理的调度算法。

## 2026-05-06：编写第一个 Go 单元测试

本阶段目标：不用手动打印状态，而是用测试代码自动验证“请求进入后，电梯会移动”。

### 本次新增文件

- `internal/elevator/system_test.go`：`System` 同步模拟逻辑的测试。

### Go 测试文件的基本规则

Go 的测试文件有几个约定：

- 文件名必须以 `_test.go` 结尾。
- 测试函数名通常以 `Test` 开头。
- 测试函数参数是 `t *testing.T`。
- 运行测试使用：

  ```bash
  go test ./...
  ```

例如：

```go
func TestStepMovesElevatorAfterRequest(t *testing.T) {
	// test code
}
```

### `testing.T` 是什么

`testing.T` 是 Go 测试框架传进来的对象。

测试里可以用它报告失败，例如：

```go
t.Fatalf("first elevator floor = %d, want 2", firstElevator.CurrentFloor)
```

`Fatalf` 的意思是：

- 当前测试失败。
- 打印错误信息。
- 立即停止这个测试函数。

### 测试 1：请求进入后电梯会移动

当前测试：

```go
func TestStepMovesElevatorAfterRequest(t *testing.T) {
	system, err := NewSystem(20, 5)
	if err != nil {
		t.Fatalf("NewSystem returned error: %v", err)
	}

	if err := system.AddRequest(4, DirectionUp, RequestKindHall); err != nil {
		t.Fatalf("AddRequest returned error: %v", err)
	}

	if err := system.Step(); err != nil {
		t.Fatalf("Step returned error: %v", err)
	}

	firstElevator := system.Elevators[0]
	if firstElevator.CurrentFloor != 2 {
		t.Fatalf("first elevator floor = %d, want 2", firstElevator.CurrentFloor)
	}
}
```

它验证的过程是：

```text
创建系统：20 层，5 部电梯
添加请求：4 楼上行
调用 Step 一次
检查 1 号电梯是否从 1 楼移动到 2 楼
```

因为当前临时调度策略会把请求分配给 1 号电梯，所以第一次 `Step()` 后，1 号电梯应该向目标楼层移动一层。

### 测试 2：到达目标楼层后开门

第二个测试把目标楼层设成 2 楼：

```go
system.AddRequest(2, DirectionUp, RequestKindHall)
system.Step()
system.Step()
```

状态变化应该是：

```text
初始：电梯在 1 楼
Step 1：电梯移动到 2 楼
Step 2：电梯发现已经到达目标楼层，方向变 idle，开门，清空目标楼层
```

所以测试会检查：

- `CurrentFloor == 2`
- `Direction == DirectionIdle`
- `DoorOpen == true`
- `TargetFloors` 已经清空

### 为什么测试能帮助学习

如果只靠手动运行程序，很容易不知道状态到底有没有变。测试代码会把预期写清楚：

```text
做了什么操作
期望状态是什么
实际状态是什么
```

一旦后续改坏了 `Step()`，测试会立刻失败，并指出哪个状态不符合预期。

## 2026-05-06：重构 API 的位置

之前提到过 API 的结构一般放在 `api` 包内，所以这次重构了一下。

这里主要是注意 Go 的包机制：

两个关键规则：

  1. 不同包之间要通过 import 才能互相使用。 main.go 属于 package main，handler.go 属于
  package api，它们是不同的包。所以 main.go 必须 import 才能调用 api 包里的函数。
  2. 只有首字母大写的名字才能被其他包访问。 当前函数叫 handleHome（小写开头），这在 Go
  里是"未导出"的，只能在 api 包内部使用。移到 internal/api/handler.go
  后需要改成大写开头，比如 HandleHome。

还有一点就是之前类似 `mux.HandleFunc("/", api.HandleHome)` 也是放在 `main.go` 里面的，但是这个也不是很符合工程规范。故抽离成 `RegisterRoutes(mux *http.ServeMux)` 函数，放在 `api` 包内。

## 2026-05-06：用 struct 模式重构 API 依赖传递

本阶段目标：handler 需要访问 `*elevator.System` 才能返回电梯状态，但 handler 函数签名是固定的 `func(w, r)`，不能加参数。解决这个"依赖如何传入 handler"的问题。

### 三种方式的对比

handler 函数签名被 `net/http` 固定死了，所以依赖不能通过参数传入。Go 里常见的三种解决方式：

**方式一：包级变量**

```go
var system *elevator.System  // 包级全局变量

func handleState(w http.ResponseWriter, r *http.Request) {
    data, _ := system.Snapshot()  // 直接引用包级变量
}
```

优点：最简单，不需要额外结构。
缺点：测试时多个测试共享同一个变量，互相干扰；依赖多了以后全局变量散落各处，不清楚谁用了谁。

**方式二：闭包**

```go
func handleState(system *elevator.System) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        data, _ := system.Snapshot()  // 闭包捕获了 system
    }
}

// 注册时
mux.HandleFunc("/api/state", handleState(system))
```

优点：比包级变量干净，依赖通过参数明确传入。
缺点：每新增一个依赖，所有工厂函数的参数列表都要膨胀，`RegisterRoutes` 也跟着膨胀。

**方式三：struct 持有依赖**

```go
type Server struct {
    System *elevator.System
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
    data, _ := s.System.Snapshot()  // 通过 receiver 访问
}
```

优点：所有 handler 共享同一个"工具箱"，新增依赖只需在 `Server` 里加字段，不影响方法签名和路由注册。
缺点：比包级变量多写一点结构代码。

### 为什么选择 struct 方式

当前项目虽然依赖不多，但后续 API 会越来越多（`/api/state`、`/api/request` 等），它们都需要 `System`。struct 方式只用写一次 `Server`，之后所有 handler 都通过 `s.System` 访问，不用每次传参。

### 修改后的 handler.go

```go
package api

import (
    "encoding/json"
    "net/http"

    "os_sp26_proj1/internal/elevator"
)

type Server struct {
    System *elevator.System
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/", s.handleHome)
    mux.HandleFunc("/api/health", s.handleHealth)
    mux.HandleFunc("/api/state", s.handleState)
}
```

关键变化：

- 新增 `Server` struct，目前只持有 `*elevator.System`。后续加配置、日志等依赖直接在这里加字段。
- 所有 handler 从普通函数变成了 `Server` 的方法（`func (s *Server) handleXxx(...)`）。
- `RegisterRoutes` 也改成方法，这样注册路由时 handler 自然能通过 `s` 访问 `System`。
- `handleState` 实现了：调用 `s.System.Snapshot()` 返回 JSON。

### 修改后的 main.go

```go
func main() {
    system, err := elevator.NewSystem(20, 5)
    if err != nil {
        log.Fatalf("failed to create elevator system: %v", err)
    }

    server := &api.Server{System: system}

    mux := http.NewServeMux()
    server.RegisterRoutes(mux)

    addr := ":8080"
    log.Printf("server listening on http://localhost%s", addr)
    log.Fatal(http.ListenAndServe(addr, mux))
}
```

现在 `main` 的职责清晰了：创建系统、创建 Server、注册路由、启动服务。具体的 API 处理逻辑全在 `api` 包里。


## 2026-05-06：设计 `POST /api/request`

本阶段目标：让前端可以通过 HTTP 请求向后端提交一个新的乘梯请求。

### API 设计

当前接口：

```text
POST /api/request
```

请求体使用 JSON：

```json
{
  "floor": 5,
  "direction": "up",
  "kind": "hall"
}
```

字段含义：

- `floor`：请求楼层。
- `direction`：请求方向，当前可用 `"up"`、`"down"`、`"idle"`。
- `kind`：请求来源，当前可用 `"hall"`、`"cabin"`。如果不传，后端默认当作 `"hall"`。

### 为什么请求里不需要传电梯序号

用户在楼层外面按按钮时，通常不知道也不应该指定哪部电梯来接。

例如用户只表达：

```text
我在 5 楼，我要上行
```

至于派哪部电梯，是调度系统的职责。

所以 `POST /api/request` 只提交乘客请求，不提交电梯编号。后端收到请求后，会先放入 `PendingRequests`，之后由 `Step()` 或调度算法分配给电梯。

如果前端传入电梯序号，就相当于前端绕过了调度算法，这不符合本项目“电梯调度系统”的目标。

### Go 如何从 HTTP 请求中读取参数

在 `internal/api/handler.go` 里，`handleRequest` 使用了一个临时结构体：

```go
var request struct {
	Floor     int                  `json:"floor"`
	Direction elevator.Direction  `json:"direction"`
	Kind      elevator.RequestKind `json:"kind"`
}
```

然后用：

```go
err := json.NewDecoder(r.Body).Decode(&request)
```

读取请求体。

这里的关键点：

- `r.Body` 是 HTTP 请求体。
- `json.NewDecoder(r.Body)` 创建一个 JSON 解码器。
- `Decode(&request)` 把 JSON 里的字段填入 Go 结构体。
- `&request` 表示传入结构体地址，这样 `Decode` 才能修改它。

如果前端发送：

```json
{
  "floor": 5,
  "direction": "up",
  "kind": "hall"
}
```

解码后 Go 里的值就是：

```go
request.Floor == 5
request.Direction == elevator.DirectionUp
request.Kind == elevator.RequestKindHall
```

### 用 `curl` 测试

启动后端：

```bash
go run ./cmd/server
```

提交请求：

```bash
curl -X POST http://localhost:8080/api/request \
  -H "Content-Type: application/json" \
  -d '{"floor":5,"direction":"up","kind":"hall"}'
```

成功时返回：

```json
{"status":"accepted"}
```

然后查看状态：

```bash
curl http://localhost:8080/api/state
```

应该能看到 `pendingRequests` 里出现刚才提交的请求。后续调用 `Step()` 或定时推进逻辑后，请求会被分配给电梯。

### JSON 编码的两种方式

在 `handleState` 和 `handleRequest` 里，返回 JSON 的方式不同：

**方式一：`w.Write(data)` — 直接写现成的字节切片**

```go
// handleState
data, err := s.System.Snapshot()
w.Header().Set("Content-Type", "application/json; charset=utf-8")
w.Write(data)
```

`Snapshot()` 返回的是 `[]byte`，也就是已经编码好的 JSON 字节。所以直接用 `w.Write` 原封不动写出去即可。

**方式二：`json.NewEncoder(w).Encode(v)` — 实时编码 Go 对象**

```go
// handleRequest
response := map[string]string{"status": "accepted"}
w.Header().Set("Content-Type", "application/json; charset=utf-8")
json.NewEncoder(w).Encode(response)
```

`response` 是一个 Go map，不是 JSON 字节。`json.NewEncoder(w).Encode(response)` 会实时把它编码成 JSON 并写入 `w`。

**区别总结：**

```text
方式一（Write）：
  []byte 数据                                     → 直接写入 w → 发给客户端
  适用于：数据已经是 JSON 了（例如从 Snapshot 拿到的）

方式二（Encode）：
  Go map/struct  → 内部 JSON 编码 → 写入 w → 发给客户端
  适用于：数据是 Go 对象，需要当场转成 JSON
```

**两种方式的对比：**

| | `w.Write(data)` | `json.NewEncoder(w).Encode(v)` |
|---|---|---|
| 输入 | `[]byte`（已经是 JSON） | 任意 Go 类型 |
| 编码步骤 | 不需要，直接写 | 需要实时编码 |
| 适合场景 | 数据已经提前编码好了 | 动态构造的 Go 对象 |
| 错误处理 | `Write` 返回 `(int, error)`，简单场景可忽略 | `Encode` 返回 `error`，建议检查 |
| 内存 | 一次性分配完整 JSON | 流式写入，但小对象差别不大 |

**为什么 `handleState` 不也用 `Encode`？**

因为 `Snapshot()` 已经做了 `json.MarshalIndent`，返回的就是 JSON。如果再 `Encode` 一次，会对 JSON 做二次编码，导致返回的内容变成带转义引号的字符串，而不是正常的 JSON 对象。

**为什么 `handleRequest` 不先用 `json.Marshal` 再 `Write`？**

可以，但多此一举：

```go
// 多余的做法：
bytes, _ := json.Marshal(response)   // 先编成 []byte
w.Write(bytes)                        // 再写出

// 更直接的做法：
json.NewEncoder(w).Encode(response)   // 编码 + 写出一步到位
```

`Encode` 在底层就是先序列化再写出，没必要中间多存一个 `bytes` 变量。

### 当前实现的限制

- `POST /api/request` 只负责添加请求，不会自动推进电梯。
- 当前还没有 HTTP API 暴露 `Step()`。
- 后续可以设计 `POST /api/step` 作为调试接口，或者让后端定时调用 `Step()`。