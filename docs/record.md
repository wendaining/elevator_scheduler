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

比如当前这个接口：`GET /api/health`

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

- 第一步：handler 暂时放 main.go，方便理解 HTTP 服务最小结构
- 第二步：出现业务 API 后，把 handler 移到 internal/api/
- 第三步：main.go 只负责组装，不负责具体业务处理

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

浏览器会自动向后端发送一个 `GET` 请求：`GET /api/health`

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
- 当前已添加临时调试接口 `POST /api/step`，用于同步模型阶段手动推进一次 `Step()`。
- 后续进入并发模型后，可以取消这个调试接口，改成后端自己定时推进。

## 2026-05-07：建立最小原生前端通信页

本阶段目标：先不用 Vue，使用原生 HTML、CSS、JavaScript 做一个很薄的前端页面，打通下面这条链路：

```text
点击楼层按钮
    ↓
JavaScript 调用 POST /api/request
    ↓
Go 后端把请求加入 PendingRequests
    ↓
JavaScript 定时调用 POST /api/step 推进模拟
    ↓
JavaScript 调用 GET /api/state 读取最新状态
    ↓
页面重新渲染电梯状态
```

这个页面是通信调试页，不是最终工业化前端。它的价值是把前端和后端通信的基本过程看清楚。闭环跑通后，再迁移到 Vue + Vite。

### 本次涉及文件

- `web/index.html`：页面结构。
- `web/style.css`：页面样式。
- `web/app.js`：创建按钮、发送请求、读取状态、更新页面。
- `internal/api/handler.go`：增加静态文件服务和临时 `POST /api/step` 调试接口。

### HTML 负责什么

`web/index.html` 只负责页面结构，不负责业务逻辑。

当前页面里有几个关键节点：

```html
<div id="floorList" class="floor-list"></div>
<div id="elevatorList" class="elevator-list"></div>
<pre id="pendingRequests" class="pending-output">[]</pre>
<p id="statusText" class="status">Connecting...</p>
```

这些节点一开始是空容器。后续 JavaScript 会找到它们，再动态填入楼层按钮、电梯卡片和待处理请求。

可以先这样理解：

```text
HTML 提供容器
CSS 控制外观
JavaScript 填充内容并处理交互
```

### CSS 负责什么

`web/style.css` 只负责视觉布局，例如：

- 页面最大宽度。
- 三栏布局。
- 面板边框和间距。
- 楼层按钮的排列。
- 电梯状态卡片的排列。

CSS 不负责请求后端，也不保存电梯状态。它只关心“显示出来长什么样”。

### JavaScript 负责什么

`web/app.js` 负责前端行为。当前最重要的函数是：

```js
createFloorButtons()
fetchState()
submitRequest(floor, direction)
advanceOneStep()
renderState(state)
refreshState()
tick()
```

#### 创建楼层按钮

```js
document.createElement("button")
```

表示创建一个新的按钮元素。

```js
upButton.addEventListener("click", () => submitRequest(floor, "up"))
```

表示给按钮绑定点击事件。用户点击按钮时，浏览器会调用 `submitRequest(floor, "up")`。

#### 读取后端状态

```js
const response = await fetch("/api/state");
const state = await response.json();
```

这会发送：

```text
GET /api/state
```

后端返回 JSON，前端把 JSON 解析成 JavaScript 对象。

#### 提交楼层请求

```js
await fetch("/api/request", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    floor,
    direction,
    kind: "hall",
  }),
});
```

这会发送：

```text
POST /api/request
```

请求体是 JSON，例如：

```json
{
  "floor": 5,
  "direction": "up",
  "kind": "hall"
}
```

这里的 `JSON.stringify(...)` 是把 JavaScript 对象转换成 JSON 字符串。

#### 渲染页面

```js
function renderState(state) {
  elevatorList.replaceChildren();

  for (const elevator of state.elevators) {
    // create elevator card
  }
}
```

`renderState` 接收后端返回的系统状态，然后重新生成电梯显示卡片。

当前做法很朴素：每次刷新时清空旧内容，再重新创建 DOM。这个方式适合调试页；以后迁移到 Vue 后，Vue 会用响应式系统帮我们更自然地更新页面。

### 为什么加临时 `POST /api/step`

当前后端还没有 goroutine 定时推动电梯运行，所以如果只提交请求，电梯不会自己移动。

为了让页面能看到状态变化，先加一个调试接口：

```text
POST /api/step
```

它会调用：

```go
s.System.Step()
```

前端每隔 800ms 调用一次：

```js
setInterval(tick, 800);
```

`tick()` 做两件事：

```text
POST /api/step   推进系统一步
GET /api/state   读取最新状态并渲染
```

这个接口是同步模拟阶段的调试工具。后续进入并发模型后，可以让后端自己定时推进，前端只负责 `GET /api/state`。

### 后端如何提供前端文件

`internal/api/handler.go` 中增加了：

```go
mux.Handle("/", http.FileServer(http.Dir("web")))
```

意思是：把 `web/` 目录作为静态文件目录。

浏览器访问：

```text
http://localhost:8080/
```

Go 会返回：

```text
web/index.html
```

页面里的：

```html
<link rel="stylesheet" href="/style.css" />
<script src="/app.js"></script>
```

会继续请求：

```text
GET /style.css
GET /app.js
```

Go 的静态文件服务会从 `web/` 目录里找到对应文件并返回。

### 当前限制

- 这是原生前端通信调试页，不是最终 Vue 前端。
- 页面没有复杂动画。
- 页面没有调度算法切换。
- 页面没有组件化。
- `POST /api/step` 是同步模拟阶段的临时调试接口。

当前只关注一个目标：看懂前端如何通过 HTTP API 驱动后端状态变化，并把状态显示出来。

## 2026-05-07：前端通信页相关问题整理

### 1. 什么是“使用原生 DOM API 和 fetch”

这里其实有两个概念：

```text
原生 DOM API
fetch
```

#### 原生 DOM API

DOM 是 Document Object Model，可以先理解为：浏览器把 HTML 页面解析成了一棵可以被 JavaScript 操作的对象树。

比如 HTML 里有：

```html
<div id="floorList"></div>
```

JavaScript 可以用：

```js
const floorList = document.querySelector("#floorList");
```

找到这个节点。

也可以用：

```js
const button = document.createElement("button");
button.textContent = "Up";
floorList.append(button);
```

创建一个新按钮，并放进页面里。

这些 `document.querySelector`、`document.createElement`、`append`、`addEventListener` 都是浏览器内置的 DOM API，不依赖 Vue、React 或其他框架，所以叫“原生 DOM API”。

#### fetch

`fetch` 是浏览器内置的 HTTP 请求 API。

例如：

```js
const response = await fetch("/api/state");
const state = await response.json();
```

它会让浏览器向后端发送：

```text
GET /api/state
```

再把后端返回的 JSON 解析成 JavaScript 对象。

所以“使用原生 DOM API 和 fetch”的意思是：

```text
不用 Vue / React
直接用浏览器内置能力创建页面元素、监听点击事件、发送 HTTP 请求
```

当前这样做是为了先看清楚最小前后端通信流程。后续迁移到 Vue 后，Vue 会帮我们更方便地管理 DOM，但 HTTP 请求和数据流本质还是类似的。

### 2. `floorCount` 后续应该从哪里来

当前 `web/app.js` 里写了：

```js
const floorCount = 20;
```

这是临时写法。它的问题是：前端和后端各自写了一份楼层数。

后端初始化时是：

```go
elevator.NewSystem(20, 5)
```

前端又写：

```js
const floorCount = 20;
```

如果以后后端改成 30 层，但前端忘记改，页面就错了。

所以更合理的是：后端作为系统状态的权威来源，前端从：

```text
GET /api/state
```

读取：

```json
{
  "floorCount": 20
}
```

然后根据后端返回的 `floorCount` 生成楼层按钮。

#### 那是否应该用 POST 表单写入后端

这取决于“楼层数是不是用户可以配置的”。

如果只是读取当前系统配置：

```text
GET /api/state
```

就够了。

如果后续希望页面提供一个配置表单，例如用户输入：

```text
楼层数：30
电梯数：6
```

然后修改后端系统配置，那确实应该设计一个写入接口，例如：

```text
PUT /api/config
```

或者简单一点：

```text
POST /api/config
```

请求体可能是：

```json
{
  "floorCount": 30,
  "elevatorCount": 6
}
```

所以这里有两个不同场景：

```text
读取当前楼层数：GET /api/state
修改系统楼层数：POST/PUT /api/config
```

当前阶段我们只是展示已有系统状态，不做系统配置页面，所以先从 `GET /api/state` 读取 `floorCount` 更合适。

### 3. `createFloorButtons()` 是不是在手写 HTML

可以这样理解：它是用 JavaScript 动态创建 HTML 元素。

手写 HTML 的方式可能是：

```html
<div class="floor-row">
  <span class="floor-label">5F</span>
  <button>Up</button>
  <button>Down</button>
</div>
```

而 `createFloorButtons()` 做的是同一件事，只是用 JS 写：

```js
const row = document.createElement("div");
row.className = "floor-row";

const label = document.createElement("span");
label.className = "floor-label";
label.textContent = `${floor}F`;

const upButton = document.createElement("button");
upButton.textContent = "Up";

row.append(label, upButton, downButton);
floorList.append(row);
```

所以它可以理解为：**用 JavaScript 生成 HTML**

为什么这么做？因为楼层有 20 层，如果手写 HTML，就要重复写 20 行楼层结构。用循环可以根据 `floorCount` 自动生成。

后续迁移到 Vue 后，这段逻辑会变成类似：

```html
<div v-for="floor in floors" :key="floor">
  ...
</div>
```

本质仍然是“根据数据生成页面元素”，只是 Vue 的写法更简洁。

### 4. `async` 和 `await` 在这里的作用

`fetch` 是异步操作，因为浏览器发送 HTTP 请求后，需要等待后端响应。

如果不用异步机制，代码不能立刻拿到后端返回值。

当前代码：

```js
async function fetchState() {
  const response = await fetch("/api/state");
  return response.json();
}
```

可以这样读：

```text
async function
  表示这个函数里会有异步操作

await fetch("/api/state")
  等待 HTTP 请求完成

response.json()
  把响应体解析成 JavaScript 对象
```

如果没有 `await`：

```js
const response = fetch("/api/state");
```

此时 `response` 不是最终的 HTTP 响应，而是一个 Promise。Promise 可以先理解为“未来才会完成的结果”。

所以：

```js
const response = await fetch("/api/state");
```

意思是：

```text
先暂停当前 async 函数
等 fetch 真的拿到响应后
再把响应赋值给 response
```

再看提交请求：

```js
async function submitRequest(floor, direction) {
  const response = await fetch("/api/request", {
    method: "POST",
    ...
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message);
  }

  await refreshState();
}
```

这里有三个异步点：

1. 等 `POST /api/request` 完成。
2. 如果失败，等 `response.text()` 读出错误信息。
3. 成功后，等 `refreshState()` 重新读取状态并渲染页面。

简单记忆：

```text
async：声明这个函数里可以使用 await
await：等待一个异步操作完成，再继续执行下一行
```

### 5. 为什么接入并发模型后，前端就不需要自己 `tick`

当前后端是同步模拟模型。也就是说，后端状态只有在有人调用：

```go
system.Step()
```

时才会推进。

如果前端只调用：

```text
POST /api/request
```

那么后端只是把请求加入 `PendingRequests`。电梯不会自动移动，因为没有任何后台逻辑在持续调用 `Step()`。

所以当前前端临时做了：

```js
setInterval(tick, 800);
```

而 `tick()` 里调用：

```text
POST /api/step
GET /api/state
```

也就是让前端承担了“定时推动系统运行”的角色。

#### 并发模型后会发生什么

后续如果后端接入 goroutine，可以让后端自己运行一个循环：

```go
go func() {
  for {
    system.Step()
    time.Sleep(800 * time.Millisecond)
  }
}()
```

这样后端自己就会每隔一段时间推进系统状态。

此时前端不需要再调用：

```text
POST /api/step
```

前端只需要定时读取状态：

```text
GET /api/state
```

也就是说，职责会变成：

```text
后端 goroutine：负责推动电梯运行
前端 setInterval：只负责定时读取并展示状态
```

这就是为什么接入并发模型后，前端不再需要执行“推进系统”的 `tick` 逻辑。更准确地说，前端可能仍然会有定时刷新函数，但它只刷新状态，不再负责让系统前进一步。

当前阶段：

```text
前端 tick = 推进后端 + 读取状态
```

并发阶段：

```text
后端 goroutine = 推进系统
前端 refresh = 读取状态
```

### 6. `mux.Handle()` 和 `mux.HandleFunc()` 的区别

它们都用于注册路由，但接收的第二个参数不同。

#### `HandleFunc`

当前 API 路由使用：

```go
mux.HandleFunc("/api/state", s.handleState)
```

`HandleFunc` 接收的是一个函数。这个函数长这样：

```go
func(w http.ResponseWriter, r *http.Request)
```

例如：

```go
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
  ...
}
```

所以：

```text
HandleFunc = 给我一个函数，我帮你把它当作 HTTP handler
```

#### `Handle`

静态文件服务使用：

```go
mux.Handle("/", http.FileServer(http.Dir("web")))
```

`Handle` 接收的不是普通函数，而是一个实现了 `http.Handler` 接口的对象。

`http.Handler` 接口大概是：

```go
type Handler interface {
  ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

`http.FileServer(http.Dir("web"))` 返回的就是一个 `http.Handler`。它内部已经知道如何处理请求、读取文件、返回文件内容。

所以：

```text
Handle = 给我一个已经实现 ServeHTTP 的 handler 对象
```

#### 为什么语法差别看起来很大

因为这两种写法面向的对象不同：

```go
mux.HandleFunc("/api/state", s.handleState)
```

这里 `s.handleState` 是我们自己写的函数。

```go
mux.Handle("/", http.FileServer(http.Dir("web")))
```

这里 `http.FileServer(...)` 是标准库帮我们创建好的 handler 对象。

可以这样记：

```text
自己写函数处理请求：HandleFunc
使用现成 handler 对象：Handle
```

事实上，`HandleFunc` 可以理解为一个便利写法。它把普通函数包装成 `http.Handler`。所以两者不是完全不同的世界，而是同一个 HTTP handler 模型下的两种入口。

### 7. 现在如何启动这个项目

当前项目还没有 Vue、npm 或 Docker，启动方式很简单：直接运行 Go 后端。

在项目根目录执行：

```bash
go run ./cmd/server
```

如果启动成功，终端会看到类似：

```text
server listening on http://localhost:8080
```

然后在浏览器打开：

```text
http://localhost:8080/
```

Go 后端会通过：

```go
mux.Handle("/", http.FileServer(http.Dir("web")))
```

把 `web/index.html`、`web/style.css`、`web/app.js` 提供给浏览器。

也就是说，现在不是直接双击打开 `web/index.html`，而是通过 Go 服务访问页面。这样页面里的：

```js
fetch("/api/state")
fetch("/api/request")
fetch("/api/step")
```

才能正确请求到同一个后端服务。

#### 用 curl 验证后端 API

查看系统状态：

```bash
curl http://localhost:8080/api/state
```

提交一个楼层请求：

```bash
curl -X POST http://localhost:8080/api/request \
  -H "Content-Type: application/json" \
  -d '{"floor":5,"direction":"up","kind":"hall"}'
```

推进系统一步：

```bash
curl -X POST http://localhost:8080/api/step
```

再次查看状态：

```bash
curl http://localhost:8080/api/state
```

可以观察 1 号电梯是否逐步移动。

#### 如何停止服务

运行 `go run ./cmd/server` 的终端会一直被后端服务占用。

停止服务时，在那个终端里按：

```text
Ctrl + C
```

#### 如果端口被占用

如果看到：

```text
listen tcp :8080: bind: address already in use
```

说明 8080 端口已经有别的程序在使用，可能是之前启动的服务没有关掉。

可以先检查是否已有项目服务在运行：

```bash
curl http://localhost:8080/api/health
```

如果能返回：

```json
{"status":"ok"}
```

说明已经有一个后端服务在跑。

#### 每次启动前建议先跑测试

```bash
go test ./...
```

如果测试通过，再启动：

```bash
go run ./cmd/server
```

当前完整启动流程可以记成：

```text
go test ./...
go run ./cmd/server
浏览器打开 http://localhost:8080/
```

### 8. 为什么现在不需要给前端注册 3000 端口

现在前端文件在：

```text
web/index.html
web/style.css
web/app.js
```

它们不是由一个独立的前端开发服务器提供的，而是由 Go 后端直接提供：

```go
mux.Handle("/", http.FileServer(http.Dir("web")))
```

所以当前只有一个服务器：

```text
Go 后端：http://localhost:8080
```

浏览器访问：

```text
http://localhost:8080/
```

时，Go 返回 `web/index.html`。

然后浏览器继续请求：

```text
http://localhost:8080/style.css
http://localhost:8080/app.js
```

Go 也会从 `web/` 目录返回对应文件。

同时，前端 JavaScript 里的 API 请求也是：

```js
fetch("/api/state")
fetch("/api/request")
fetch("/api/step")
```

这些请求也会发到同一个地址：

```text
http://localhost:8080
```

所以当前结构是：

```text
同一个 8080 端口
  ├── 返回前端静态文件：/、/style.css、/app.js
  └── 返回后端 API：/api/state、/api/request、/api/step
```

因此不需要再给前端单独开一个 3000 端口。

#### 那为什么很多项目有 3000 端口

很多现代前端项目会使用 Vite、React、Vue 等开发服务器。比如 Vue + Vite 通常会启动：

```text
前端开发服务器：http://localhost:5173
```

React 旧项目常见：

```text
前端开发服务器：http://localhost:3000
```

这种情况下会有两个服务器：

```text
前端开发服务器：http://localhost:3000 或 5173
Go 后端服务器：http://localhost:8080
```

前端页面从 3000/5173 加载，但 API 在 8080，所以会出现跨端口请求：

```text
浏览器页面：http://localhost:3000
请求 API：http://localhost:8080/api/state
```

这时就需要处理代理或 CORS。

#### 当前为什么先不用这种方式

当前阶段只是原生 HTML/CSS/JS 通信调试页，没有 npm、Vite、Vue，也没有构建步骤。

直接让 Go 同时提供页面和 API，有几个好处：

- 启动命令只有一个：`go run ./cmd/server`
- 没有跨域问题
- 不需要 npm
- 更容易看清楚 HTTP API 和页面之间的关系

后续迁移到 Vue + Vite 后，开发阶段可能会变成：

```text
前端：npm run dev       例如 http://localhost:5173
后端：go run ./cmd/server  http://localhost:8080
```

到那时再学习前端 dev server、代理和跨域会更合适。

### 9. 什么是“独立前端开发服务器”

“独立前端开发服务器”可以拆开理解：

```text
独立：它和 Go 后端不是同一个进程
前端：它主要负责提供 HTML、CSS、JavaScript 等前端资源
开发服务器：它主要用于开发阶段，不一定是最终部署方式
```

当前项目现在只有一个服务器：

```text
go run ./cmd/server
```

这个 Go 进程同时做两件事：

```text
1. 提供后端 API
   /api/state
   /api/request
   /api/step

2. 提供前端静态文件
   /
   /style.css
   /app.js
```

所以当前结构是：

```text
浏览器
  ↓
http://localhost:8080
  ↓
Go 服务器
  ├── 返回 web/index.html
  ├── 返回 web/style.css
  ├── 返回 web/app.js
  └── 处理 /api/... 请求
```

这不是独立前端开发服务器，因为前端文件是由 Go 后端顺便提供的。

#### 独立前端开发服务器长什么样

如果后续使用 Vue + Vite，通常会启动一个前端开发服务器：

```bash
npm run dev
```

它可能输出：

```text
Local: http://localhost:5173/
```

这时会有两个进程：

```text
进程 1：Vite 前端开发服务器
地址：http://localhost:5173
职责：提供 Vue 页面、热更新、前端构建能力

进程 2：Go 后端服务器
地址：http://localhost:8080
职责：提供 /api/state、/api/request 等后端 API
```

浏览器访问页面时，访问的是：

```text
http://localhost:5173
```

页面里的 API 请求再转发或发送到：

```text
http://localhost:8080/api/state
```

这就是“独立前端开发服务器”：

```text
前端页面由一个服务器提供
后端 API 由另一个服务器提供
```

#### 为什么前端框架需要开发服务器

Vue / React 项目通常不是浏览器直接读取一个简单 `app.js` 就结束了。

它们会有：

- `.vue` 单文件组件
- 多个 JavaScript 模块
- npm 依赖
- CSS 预处理或模块化
- 开发时热更新
- 打包构建

浏览器不能直接理解所有开发阶段的文件格式和模块组织，所以 Vite 这样的工具会在开发时帮忙处理：

```text
读取源代码
解析模块依赖
把浏览器能运行的 JS 发给浏览器
代码变化时通知浏览器自动刷新
```

所以它需要启动一个开发服务器。

#### 当前 Go 静态文件服务和 Vite 开发服务器的区别

| | 当前 Go 静态文件服务 | Vite 前端开发服务器 |
|---|---|---|
| 启动命令 | `go run ./cmd/server` | `npm run dev` |
| 常见端口 | `8080` | `5173` 或其他 |
| 是否独立于后端 | 否，和后端同一个 Go 进程 | 是，单独一个前端进程 |
| 主要职责 | 返回已有的 HTML/CSS/JS 文件 | 编译、转换、热更新前端源码 |
| 是否需要 npm | 不需要 | 需要 |
| 是否会涉及跨端口 API | 不会 | 通常会 |

#### 一句话总结

当前阶段：

```text
Go 一个服务器同时提供页面和 API
```

Vue/Vite 阶段：

```text
Vite 服务器提供前端页面
Go 服务器提供后端 API
```

所以现在不需要 3000/5173 端口；等引入 Vue + Vite 后，才会出现独立前端开发服务器这个概念。

### 10. Vue/Vite 能不能完全由浏览器处理，是否必须有服务器托管

这个问题要区分两个阶段：

```text
开发阶段
部署阶段
```

#### 开发阶段：需要 Vite 开发服务器

Vue/Vite 项目的源码通常不是浏览器可以直接完整运行的形态。

开发源码里可能有：

```text
.vue 单文件组件
import/export 模块
npm 依赖
组件拆分
开发热更新
```

浏览器不能直接理解 `.vue` 文件，也不会帮我们解析 npm 依赖、做热更新、处理开发时的模块转换。

所以开发时通常要运行：

```bash
npm run dev
```

这会启动 Vite 开发服务器，例如：

```text
http://localhost:5173
```

Vite 开发服务器负责：

```text
读取 Vue 源码
转换成浏览器能运行的 JavaScript / CSS
处理模块依赖
提供热更新
把页面发给浏览器
```

所以在开发阶段，Vue/Vite 源码不能像当前 `web/app.js` 一样直接丢给浏览器处理。

#### 部署阶段：可以是静态文件

但 Vue/Vite 项目最终可以打包：

```bash
npm run build
```

打包后通常会得到：

```text
dist/
  index.html
  assets/*.js
  assets/*.css
```

这些就是普通静态文件。

部署时可以由很多东西托管：

```text
Go 后端的 http.FileServer
Nginx
GitHub Pages
Vercel
Netlify
任意静态文件服务器
```

也就是说，部署阶段不一定需要 Vite 开发服务器。

可以是：

```text
浏览器
  ↓
Go / Nginx 返回 dist/index.html 和 JS/CSS
  ↓
JS 运行后 fetch 后端 API
```

#### 更准确的理解

不应该说：Vue/Vite 不能静态托管

更准确的说法是：Vue 源码在开发阶段不能直接给浏览器运行，需要 Vite 转换；Vue build 之后的产物是普通静态文件，可以静态托管。

对应到本项目：

当前原生阶段：

```text
web/index.html
web/app.js
web/style.css
  ↓
Go 直接托管
```

未来 Vue 开发阶段：

```text
Vue 源码
  ↓ npm run dev
Vite 开发服务器 http://localhost:5173
```

未来最终部署阶段：

```text
Vue 源码
  ↓ npm run build
dist/ 静态文件
  ↓
Go 或 Nginx 托管
```

一句话总结：

```text
Vue/Vite 的开发源码不能直接完全由浏览器处理；
但 build 之后的产物可以完全当作静态文件托管。
```

### 11. npm、`npm run dev`、`npm run build` 是什么

这个问题本质上是在问：现代前端项目是怎么组织、运行和打包的。

当前项目的原生前端很简单：

```text
web/index.html
web/style.css
web/app.js
```

浏览器可以直接运行这些文件，所以不需要 npm。

但 Vue / React 项目通常会复杂很多，需要 npm 来管理依赖和运行脚本。

#### npm 是什么

npm 可以先理解为 JavaScript / Node.js 生态里的包管理工具。

它主要做两类事情：

```text
1. 管理依赖
2. 运行项目脚本
```

例如 Vue 项目可能依赖：

```text
vue
vite
@vitejs/plugin-vue
```

这些依赖不是浏览器自带的，需要 npm 下载到本地项目里。

常见命令：

```bash
npm install
```

意思是：根据 `package.json` 里的依赖列表，把需要的包下载到 `node_modules/` 目录。

#### `package.json` 是什么

`package.json` 是前端项目的配置文件，可以类比 Go 项目里的 `go.mod`，但它不只记录依赖，也记录脚本命令。

一个简化的 Vue + Vite `package.json` 可能长这样：

```json
{
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "vue": "^3.0.0"
  },
  "devDependencies": {
    "vite": "^5.0.0",
    "@vitejs/plugin-vue": "^5.0.0"
  }
}
```

这里最重要的是：

```json
"scripts": {
  "dev": "vite",
  "build": "vite build"
}
```

它定义了项目里可以运行哪些命令。

#### `npm run dev` 是什么

当执行：

```bash
npm run dev
```

npm 会去 `package.json` 里找：

```json
"dev": "vite"
```

然后实际执行：

```bash
vite
```

也就是说：

```text
npm run dev
  ↓
读取 package.json 的 scripts.dev
  ↓
执行 vite
  ↓
启动 Vite 前端开发服务器
```

它通常会输出：

```text
Local: http://localhost:5173/
```

这个服务器主要用于开发阶段，提供：

- Vue 文件转换
- JavaScript 模块处理
- npm 依赖解析
- 热更新
- 本地页面访问地址

所以：

```text
npm run dev = 启动前端开发环境
```

它不是最终部署命令。

#### `npm run build` 是什么

当执行：

```bash
npm run build
```

npm 会去 `package.json` 里找：

```json
"build": "vite build"
```

然后实际执行：

```bash
vite build
```

这个命令会把 Vue/Vite 源码打包成浏览器可以直接运行的静态文件。

通常输出到：

```text
dist/
  index.html
  assets/*.js
  assets/*.css
```

所以：

```text
npm run build = 生成最终部署用的静态文件
```

开发时看页面用：

```bash
npm run dev
```

准备部署时用：

```bash
npm run build
```

#### `npm run preview` 是什么

有些 Vite 项目还有：

```bash
npm run preview
```

它通常对应：

```json
"preview": "vite preview"
```

它的作用是：在本地预览 `npm run build` 生成的 `dist/` 结果。

区别是：

```text
npm run dev
  运行开发源码，支持热更新

npm run build
  生成 dist 静态文件

npm run preview
  本地预览 dist 静态文件
```

#### dependencies 和 devDependencies

`package.json` 里通常有两类依赖：

```json
"dependencies": {}
"devDependencies": {}
```

可以先这样理解：

```text
dependencies
  项目运行时需要的依赖
  例如 vue

devDependencies
  开发和构建时需要的依赖
  例如 vite、测试工具、代码格式化工具
```

对初学阶段来说，不需要过度纠结。先知道：

```bash
npm install
```

会把这些依赖都装好。

#### `node_modules` 是什么

`node_modules/` 是 npm 下载依赖后存放的目录。

它通常很大，而且不应该提交到 Git。

前端项目通常会把它写进 `.gitignore`：

```text
node_modules/
```

别人拿到项目后，只需要运行：

```bash
npm install
```

就可以根据 `package.json` 和 `package-lock.json` 重新安装依赖。

#### `package-lock.json` 是什么

`package-lock.json` 用来记录依赖的精确版本。

`package.json` 里可能写的是：

```json
"vite": "^5.0.0"
```

这表示允许安装某个范围内的版本。

而 `package-lock.json` 会记录实际安装的精确版本，例如：

```text
vite 5.2.10
```

这样团队里不同人安装依赖时，版本更一致。

#### 和 Go 项目的类比

可以粗略类比：

| Go | 前端 npm |
|---|---|
| `go.mod` | `package.json` |
| `go.sum` | `package-lock.json` |
| `go test ./...` | `npm test`，如果项目配置了 |
| `go run ./cmd/server` | `npm run dev`，用于启动前端开发服务器 |
| `go build` | `npm run build`，用于生成构建产物 |

这个类比不是完全一一对应，但有助于建立直觉。

`go run`
- 临时编译
- 立即运行
- 不保留最终可执行文件

`go build`
- 真正生成可执行文件
- 不自动运行

#### 对本项目意味着什么

当前阶段：

```text
原生 HTML/CSS/JS
Go 直接托管 web/
不需要 npm
```

未来 Vue 阶段：

```text
需要 package.json
需要 npm install
开发时运行 npm run dev
后端仍运行 go run ./cmd/server
```

未来部署阶段：

```text
npm run build 生成 dist/
Go 或 Nginx 托管 dist/
```

所以可以记成：

```text
npm install     安装前端依赖
npm run dev     启动前端开发服务器
npm run build   生成可部署的静态文件
npm run preview 本地预览构建产物
```

当前还没进入 Vue/Vite 阶段，所以先不用 npm。等原生通信闭环理解清楚后，再引入 npm 会更容易。

### 12. 什么是 Vite，什么是 Nginx

这两个名字经常一起出现在前端项目里，但它们不是同一类东西。

可以先粗略记成：

```text
Vite：前端开发和构建工具
Nginx：生产环境里常用的 Web 服务器 / 反向代理服务器
```

#### Vite 是什么

Vite 是一个现代前端开发工具。

在 Vue 项目里，它通常负责两件事：

```text
开发阶段：启动前端开发服务器
构建阶段：把源码打包成静态文件
```

开发阶段运行：

```bash
npm run dev
```

如果 `package.json` 里写的是：

```json
"dev": "vite"
```

那实际启动的就是 Vite 开发服务器。

它会提供：

- 本地访问地址，例如 `http://localhost:5173`
- `.vue` 文件转换
- JavaScript 模块解析
- npm 依赖处理
- 热更新

热更新的意思是：你改了前端代码，浏览器页面可以快速更新，不需要你手动完整刷新或重新启动服务。

构建阶段运行：

```bash
npm run build
```

如果 `package.json` 里写的是：

```json
"build": "vite build"
```

Vite 会把 Vue 源码打包成：

```text
dist/
  index.html
  assets/*.js
  assets/*.css
```

这些 `dist/` 文件就是可以部署的静态文件。

所以 Vite 不是后端业务服务器。它不负责电梯调度，不负责数据库，也不负责 Go API。它主要负责前端源码的开发体验和构建。

#### Nginx 是什么

Nginx 是一个高性能 Web 服务器，也经常用作反向代理服务器。

它常见用途包括：

```text
1. 托管静态文件
2. 把请求转发给后端服务
3. 处理 HTTPS
4. 做负载均衡
```

例如部署 Vue 项目时，可以让 Nginx 托管 Vite build 出来的 `dist/`：

```text
浏览器
  ↓
Nginx
  ↓
返回 dist/index.html、assets/*.js、assets/*.css
```

同时，Nginx 也可以把 API 请求转发给 Go 后端：

```text
浏览器请求 /api/state
  ↓
Nginx
  ↓
转发给 Go 后端 localhost:8080/api/state
```

这种“把请求转发给另一个服务”的行为叫反向代理。

#### Vite 和 Nginx 的区别

| | Vite | Nginx |
|---|---|---|
| 主要阶段 | 开发、构建 | 部署、生产运行 |
| 常见命令 | `npm run dev`、`npm run build` | `nginx` 或系统服务 |
| 面向对象 | 前端源码 | HTTP 请求和静态文件 |
| 是否处理 `.vue` 文件 | 是 | 否 |
| 是否负责热更新 | 是 | 否 |
| 是否适合生产托管静态文件 | 一般不用 | 是 |
| 是否可反向代理 Go API | 开发阶段可用 proxy | 生产环境常用 |

可以这样理解：

```text
Vite 帮开发者把前端源码变成浏览器能运行的东西。
Nginx 帮服务器把已经构建好的文件和请求高效地提供给用户。
```

#### 和当前项目的关系

当前阶段没有 Vite，也没有 Nginx。

现在是：

```text
浏览器
  ↓
Go 后端 localhost:8080
  ├── 返回 web/index.html、web/app.js、web/style.css
  └── 处理 /api/state、/api/request、/api/step
```

未来 Vue 开发阶段可能是：

```text
浏览器
  ↓
Vite localhost:5173
  ↓
Vue 页面
  ↓
请求 Go 后端 localhost:8080/api/...
```

未来部署阶段可能是：

```text
浏览器
  ↓
Nginx
  ├── 返回 dist/ 静态文件
  └── 把 /api/... 转发给 Go 后端
```

也可以不用 Nginx，直接让 Go 托管 `dist/`：

```text
浏览器
  ↓
Go 后端
  ├── 返回 dist/ 静态文件
  └── 处理 /api/...
```

对于课程项目来说，后者更简单；Nginx 更像是生产环境或部署加分项。

一句话总结：

```text
Vite 是前端开发/构建工具；
Nginx 是部署时常用的 Web 服务器/反向代理。
```

#### 为什么需要 Nginx，直接 Go 托管不行吗

你的直觉是对的：对于当前课程项目，直接让 Go 后端托管前端静态文件和后端 API 完全可以。

现在这种方式是：

```text
浏览器
  ↓
Go 后端
  ├── 返回前端文件
  └── 处理 /api/...
```

它的优点是：

- 结构简单。
- 只需要启动一个进程。
- 没有跨域问题。
- 不需要额外学习 Nginx 配置。
- 很适合课程项目、原型项目、小型自用项目。

所以当前阶段不需要 Nginx。

那为什么很多生产环境还会用 Nginx？主要是因为生产部署会遇到更多工程问题。

#### 1. Nginx 很擅长提供静态文件

前端 build 后的文件通常是：

```text
dist/index.html
dist/assets/*.js
dist/assets/*.css
dist/assets/*.png
```

Nginx 提供这些静态文件非常成熟、高效，而且配置缓存策略很方便。

例如可以让浏览器长期缓存带 hash 的 JS/CSS 文件：

```text
assets/app.8f3a2.js
assets/style.91bc.css
```

这样用户第二次访问时，很多文件不需要重新下载。

Go 当然也能做静态文件服务，但 Nginx 在这类通用 Web 服务能力上更专业。

#### 2. Nginx 常用来处理 HTTPS

正式网站通常需要 HTTPS：

```text
https://example.com
```

Nginx 可以负责：

- TLS 证书配置
- HTTPS 握手
- HTTP 自动跳转 HTTPS
- 证书续期配合

这样 Go 后端可以只专注业务逻辑，不直接处理 HTTPS 细节。

#### 3. Nginx 可以做反向代理

生产环境里常见结构是：

```text
浏览器
  ↓
Nginx
  ├── /          返回前端静态文件
  └── /api/...   转发给 Go 后端
```

Go 后端可能只监听本机端口：

```text
localhost:8080
```

外部用户看不到 Go 端口，只访问：

```text
https://example.com
```

Nginx 决定哪些请求返回前端文件，哪些请求转发给 Go。

#### 4. Nginx 可以统一管理多个服务

如果以后一个服务器上有多个服务：

```text
/api/...       Go 后端
/admin/...     管理后台
/static/...    静态资源
/docs/...      文档站点
```

Nginx 可以作为统一入口，把不同路径分发给不同服务。

这时让每个后端程序都直接暴露给外部就会比较混乱。

#### 5. Nginx 可以做限流、压缩、日志等通用工作

Nginx 还常用于：

- 请求日志
- gzip / brotli 压缩
- 请求大小限制
- 简单限流
- 负载均衡
- 健康检查

这些都是通用 Web 服务能力，不一定应该写进业务后端里。

#### 什么时候不用 Nginx

以下场景完全可以不用 Nginx：

```text
课程项目
本地开发
小型 demo
单机部署
Go 后端本身已经能满足静态文件和 API 服务
```

比如本项目当前阶段：

```text
go run ./cmd/server
浏览器打开 http://localhost:8080/
```

这就足够了。

#### 什么时候考虑 Nginx

以下场景可以考虑 Nginx：

```text
要部署到公网
要配置 HTTPS
要高效托管前端 build 产物
要把多个后端服务统一到一个域名
要做反向代理、缓存、压缩、限流
```

所以结论是：Go 后端托管一切没有问题，尤其适合当前项目；Nginx 不是必须的，而是生产部署中常见的工程工具。

> wdn注：这里其实没太看懂... 感觉是没有实际操作导致的，什么反向代理都看不懂，之后系统学习一下吧，先得补网络的知识。

### 13. 前端如何区分 Hall 请求和 Cabin 请求

之前 `web/app.js` 里提交请求时写死了：

```js
kind: "hall"
```

这意味着前端只能提交楼层外部请求，也就是人在电梯外面按“上行 / 下行”按钮。

但电梯项目里其实有两类请求：

```text
Hall 请求
  来自楼层外部按钮
  用户表达：我在某一层，我要上行或下行

Cabin 请求
  来自电梯内部按钮
  用户表达：我已经在电梯里，我要去某一层
```

所以前端现在增加了一个请求类型切换：

```html
<button id="hallModeButton">Hall</button>
<button id="cabinModeButton">Cabin</button>
```

对应的 JavaScript 状态是：

```js
let currentRequestKind = "hall";
```

#### Hall 模式

Hall 模式下，每一层显示：

```text
楼层标签
Up 按钮
Down 按钮
```

点击 5 楼 Up，会提交：

```json
{
  "floor": 5,
  "direction": "up",
  "kind": "hall"
}
```

这表示：

```text
有人在 5 楼外面按了上行按钮
```

#### Cabin 模式

Cabin 模式下，每一层只显示：

```text
楼层标签
Select 按钮
```

点击 12 楼 Select，会提交：

```json
{
  "floor": 12,
  "direction": "idle",
  "kind": "cabin"
}
```

这里 `direction` 传 `"idle"` 是当前模型下的临时处理。

原因是：电梯内部按钮本质上只表达“我要去某一层”，不表达“我要上行还是下行”。但是当前后端 `Request` 结构体仍然要求有 `Direction` 字段，所以先用 `DirectionIdle` 表示“不关心方向”。

#### 改了哪些前端代码

`web/app.js` 中新增：

```js
let currentRequestKind = "hall";
```

以及：

```js
function setRequestKind(kind) {
  currentRequestKind = kind;

  hallModeButton.classList.toggle("active", kind === "hall");
  cabinModeButton.classList.toggle("active", kind === "cabin");

  createFloorButtons();
}
```

这段代码做了三件事：

```text
1. 记录当前模式
2. 更新按钮选中样式
3. 重新生成楼层按钮
```

`submitRequest` 也从原来的：

```js
submitRequest(floor, direction)
```

改成：

```js
submitRequest(floor, direction, kind)
```

这样前端不会再硬编码 `"hall"`，而是根据当前模式提交不同请求类型。

#### 当前 Cabin 模式的限制

真正的电梯内部按钮通常应该属于某一部电梯，例如：

```text
2 号电梯内部按下 12 楼
```

这需要请求里带上电梯编号，例如：

```json
{
  "elevatorId": 2,
  "floor": 12,
  "kind": "cabin"
}
```

但当前后端模型还没有 `ElevatorID` 字段，`AddRequest` 也没有按某部电梯添加内部目标楼层的逻辑。

所以当前 Cabin 模式只是一个过渡版本：

```text
前端可以提交 cabin 类型
后端仍然把它放入统一 PendingRequests 队列
临时调度策略仍然分配给 1 号电梯
```

后续如果要更真实，可以改模型：

```go
type Request struct {
  Floor      int
  Direction  Direction
  Kind       RequestKind
  ElevatorID int
}
```

或者单独设计：

```text
POST /api/elevators/{id}/requests
```

这会比当前版本更接近真实电梯内部按钮。

#### 为什么当前先这样做

当前目标还是“通信调试页”，不是最终电梯 UI。

所以这次改动优先保证：

```text
能看到 hall/cabin 两种请求的区别
能理解前端状态如何影响请求体
不一次性改太多后端模型
```

后续进入更完整的调度和前端阶段时，再把 Cabin 请求改成绑定具体电梯会更合适。

## 2026-05-07：调度算法 skeleton 和设计思路

本阶段目标：把之前写在 `system.go` 里的临时调度逻辑迁到 `scheduler.go`，并建立一个能支持多种算法切换的基础结构。

### 1. 之前的基础调度算法是否应该迁到 `scheduler.go`

应该迁。

之前 `system.go` 里有类似这样的职责：

```text
Step()
  ├── 分配请求给电梯
  └── 推进每部电梯移动
```

其中“推进电梯移动”属于系统运行逻辑，继续放在 `system.go` 可以。

但“请求分配给哪部电梯”属于调度算法，应该放到 `scheduler.go`。

所以现在职责拆成：

```text
system.go
  NewSystem
  AddRequest
  SetScheduler
  Snapshot
  Step
  stepElevator

scheduler.go
  Scheduler interface
  FirstAvailableScheduler
  NearestIdleScheduler
  NewScheduler
```

这样以后修改调度算法时，不需要反复改 `Step()` 主流程。

### 2. 如何为不同算法定义统一接口

当前定义了：

```go
type Scheduler interface {
	Name() string
	Assign(system *System) bool
}
```

这个接口要求每个调度算法都提供两个方法：

```text
Name()
  返回算法名称，例如 "first-available" 或 "nearest-idle"

Assign(system *System)
  尝试把 PendingRequests 里的请求分配给某部电梯
```

返回值 `bool` 表示：

```text
true   本次确实分配了一个请求
false  没有请求可分配，或当前没有合适电梯
```

这样 `System.Step()` 就不需要知道具体算法细节：

```go
s.scheduler.Assign(s)
```

它只知道“调用当前调度器，让它尝试分配请求”。

这就是接口的价值：

```text
System 依赖统一接口
不同算法实现同一个接口
后续切换算法时，Step 不用改
```

### 3. 不同调度算法是否应该放不同 Go 文件

可以，但不一定一开始就要拆。

当前算法还很少，所以先放在：

```text
internal/elevator/scheduler.go
```

是合理的。

等算法变多以后，可以拆成：

```text
internal/elevator/scheduler.go          放 Scheduler interface 和 NewScheduler
internal/elevator/scheduler_first.go    放 FirstAvailableScheduler
internal/elevator/scheduler_nearest.go  放 NearestIdleScheduler
internal/elevator/scheduler_scan.go     放 SCAN 算法
```

注意：这些文件只要都写：

```go
package elevator
```

它们就仍然属于同一个 Go 包，可以互相访问同一个包里的类型和函数。

所以拆文件只是为了阅读和维护，不是为了创建新的模块边界。

### 当前已有的调度器

#### `FirstAvailableScheduler`

这是从之前 `system.go` 迁出来的最小策略：

```text
只看 1 号电梯
如果 1 号电梯空闲
就把最早的请求分配给它
```

代码里对应：

```go
type FirstAvailableScheduler struct{}
```

这个算法很简单，但适合保留，因为它可以作为“最小可运行调度器”。

#### `NearestIdleScheduler`

这是“最近空闲电梯优先”的基础版本：

```text
取最早的 PendingRequest
遍历所有电梯
跳过忙碌或紧急停止的电梯
计算每部可用电梯到请求楼层的距离
选择距离最小的电梯
把请求楼层加入它的 TargetFloors
```

当前已经可运行，但还留了一个适合继续思考的 TODO：

```text
如果两部电梯距离一样，应该如何决策？
```

可以选择：

```text
编号小的优先
目标楼层更少的优先
当前方向更合适的优先
保持先遇到谁就选谁
```

当前版本采用的是“先遇到谁就选谁”，因为 `distance < bestDistance` 才更新。

### `NewScheduler` 是什么

当前有：

```go
func NewScheduler(name string) (Scheduler, error)
```

它负责根据字符串创建具体调度器：

```go
NewScheduler("first-available")
NewScheduler("nearest-idle")
```

这为后续 API 切换算法做准备。

例如以后可以设计：

```text
POST /api/scheduler
```

请求体：

```json
{
  "name": "nearest-idle"
}
```

后端调用：

```go
system.SetScheduler("nearest-idle")
```

### `System` 里新增了什么

`System` 里新增：

```go
SchedulerName string `json:"schedulerName"`
scheduler Scheduler
```

两个字段职责不同：

```text
SchedulerName
  暴露给 API 和前端，用于显示当前调度算法名称

scheduler
  真正执行调度逻辑的对象
  小写字段，不会被 JSON 暴露
```

前端现在会显示：

```text
Scheduler: first-available
```

这来自 `GET /api/state` 返回的 `schedulerName`。

### `Step()` 现在怎么工作

现在的 `Step()` 主流程变成：

```text
如果没有 scheduler，返回错误
调用当前 scheduler.Assign(s)，尝试分配请求
遍历所有电梯，推进每部电梯移动一步
```

也就是：

```go
s.scheduler.Assign(s)

for i := range s.Elevators {
	stepElevator(&s.Elevators[i])
}
```

这样调度算法和电梯运动逻辑分开了。

### 留给我继续练习的填空

当前最适合继续练习的是：

1. 给 `NearestIdleScheduler` 写测试。

   构造一个系统，让不同电梯在不同楼层，然后添加请求，检查请求是否分配给最近的空闲电梯。

2. 思考距离相同的 tie-breaker。

   例如两部电梯距离都为 2，该选谁？

3. 后续加 API 切换调度器。

   例如：

   ```text
   POST /api/scheduler
   ```

   让前端可以切换 `"first-available"` 和 `"nearest-idle"`。

当前不要急着实现 FCFS 或 SCAN。先把 `Scheduler interface -> 两个算法 -> 测试 -> API 切换` 这条线走通更重要。

### `Assign()` 到底是干嘛的

在 `internal/elevator/system.go` 里，`Step()` 现在有一行：

```go
s.scheduler.Assign(s)
```

这行的意思是：

```text
让当前调度算法尝试从 PendingRequests 里取出一个请求，
并把它分配给某一部电梯。
```

也就是说，`Assign()` 的职责不是让电梯移动。它只负责“派单”。

电梯真正移动是在后面的代码里：

```go
for i := range s.Elevators {
	stepElevator(&s.Elevators[i])
}
```

所以 `Step()` 可以拆成两步理解：

```text
第 1 步：Assign 负责分配请求
第 2 步：stepElevator 负责让电梯移动一层或开门
```

#### 一个具体例子

假设当前系统状态是：

```text
PendingRequests:
  [{ floor: 5, direction: "up", kind: "hall" }]

1 号电梯:
  currentFloor = 1
  targetFloors = []
```

调用：

```go
s.scheduler.Assign(s)
```

之后，调度器可能会做两件事：

```text
1. 从 PendingRequests 中取出最早的请求
2. 把请求楼层 5 加入某部电梯的 TargetFloors
```

如果当前使用的是 `FirstAvailableScheduler`，结果大概是：

```text
PendingRequests:
  []

1 号电梯:
  currentFloor = 1
  targetFloors = [5]
```

注意：此时电梯还没有移动。只是多了一个目标楼层。

接下来 `Step()` 会继续执行：

```go
stepElevator(&s.Elevators[0])
```

这时电梯才会从 1 楼移动到 2 楼：

```text
1 号电梯:
  currentFloor = 2
  direction = "up"
  targetFloors = [5]
```

所以一次完整的 `Step()` 里，状态变化是：

```text
Assign:
  PendingRequests -> Elevator.TargetFloors

stepElevator:
  Elevator.CurrentFloor 发生变化
```

#### 为什么 `Assign()` 返回 bool

当前定义是：

```go
Assign(system *System) bool
```

返回值表示这次有没有真的分配请求：

```text
true
  本次成功把一个请求分配给某部电梯

false
  没有请求可分配，或者暂时没有合适的电梯
```

例如这些情况会返回 `false`：

```text
PendingRequests 是空的
所有电梯都在忙
所有电梯都处于 EmergencyStop
```

#### 为什么不直接在 `Step()` 里写调度逻辑

如果直接写在 `Step()` 里，代码可能会变成：

```go
func (s *System) Step() error {
  // 最近电梯算法
  // FCFS 算法
  // SCAN 算法
  // 电梯移动逻辑
}
```

这样 `Step()` 会越来越长，而且以后切换算法会很麻烦。

现在的结构是：

```text
Step()
  调用 scheduler.Assign(s)
  调用 stepElevator(...)
```

不同算法只需要各自实现：

```go
Assign(system *System) bool
```

例如：

```text
FirstAvailableScheduler.Assign
NearestIdleScheduler.Assign
ScanScheduler.Assign
```

这样 `Step()` 的主流程就稳定了。

#### 一句话总结

`Assign()` 负责把 `System` 的 `PendingRequests` 分配给一个**特定的电梯**的 `targetFloors[]`，之后 `system.go` 里面的 `stepElevator()` 才会根据这个去移动电梯，**这一步就和调度算法无关了**。

### 如何判断一部电梯是否空闲

当前代码里，判断一部电梯是否可以接收新请求，主要看：

```go
len(e.TargetFloors) == 0
```

也就是当前实现里的：

```go
func canAcceptRequest(e Elevator) bool {
	return !e.EmergencyStop && len(e.TargetFloors) == 0
}
```

这个判断比单纯看：

```go
e.Direction == DirectionIdle
```

更可靠。

#### 为什么不只看 `DirectionIdle`

`DirectionIdle` 表示电梯当前“不在移动”。但“不在移动”不一定等于“没有任务”。

例如后续可能出现这种状态：

```text
Direction = idle
DoorOpen = true
TargetFloors = [8]
```

这表示电梯正在某层开门停靠，但后面还有 8 楼这个目标。它虽然当前方向是 `idle`，但并不是真的空闲。

也可能出现：

```text
Direction = idle
TargetFloors = [5]
```

这表示电梯刚被分配了目标楼层，但还没来得及在下一次 `Step()` 里移动。它也不是空闲。

所以：

```text
DirectionIdle 只能说明当前没有移动
不能保证没有任务
```

#### 为什么看 `TargetFloors`

`TargetFloors` 表示这部电梯还有没有待完成任务。

```text
len(TargetFloors) == 0
  没有目标楼层，当前没有任务

len(TargetFloors) > 0
  还有目标楼层，正在服务任务或即将服务任务
```

所以“是否空闲”的核心应该是：

```go
len(e.TargetFloors) == 0
```

同时还要排除紧急停止：

```go
!e.EmergencyStop
```

因为紧急停止的电梯即使没有目标楼层，也不应该接新任务。

#### 是否要看 `DoorOpen`

当前简单模型里先不看 `DoorOpen`。

如果出现：

```text
DoorOpen = true
TargetFloors = []
```

它是否能接新请求，有两种解释：

```text
简化模型：可以接，因为没有任务
更真实模型：等门关了再接
```

当前阶段为了保持调度逻辑简单，先使用：

```go
!e.EmergencyStop && len(e.TargetFloors) == 0
```

后续如果想更真实，可以改成：

```go
func (e Elevator) IsAvailable() bool {
	return !e.EmergencyStop &&
		!e.DoorOpen &&
		len(e.TargetFloors) == 0
}
```

#### 当前结论

当前阶段判断电梯是否空闲，推荐使用：

```go
func canAcceptRequest(e Elevator) bool {
	return !e.EmergencyStop && len(e.TargetFloors) == 0
}
```

因为 `TargetFloors` 比 `Direction` 更能表达“这部电梯有没有任务”。

### 2026-05-07 实现 FCFS 和 SCAN

FCFS：从 `PendingRequests` 里面取最新的请求，再取序号最小的空闲电梯。

### 2026-05-07：把 SCAN 升级为“顺路请求可追加”

之前的 SCAN skeleton 更像是“只给空闲电梯分配一个目标”。这可以跑通算法接口，但不够像真实电梯：真实电梯在已经上行或下行时，通常会顺路接同方向请求，而不是等当前目标完成后才重新调度。

这次把 `internal/elevator/SCANScheduler.go` 改成两段逻辑：

```text
1. 先尝试把请求追加给正在运行、且方向合适的电梯
2. 如果没有任何电梯能顺路接，再把请求分配给空闲电梯
```

#### 什么叫“顺路”

例如 1 号电梯现在在 3 楼，长期扫描方向是上行：

```text
CurrentFloor = 3
ScanDirection = up
TargetFloors = [10]
```

此时如果来了一个 6 楼上行 hall 请求：

```text
Floor = 6
Direction = up
Kind = hall
```

这个请求就在电梯当前扫描方向上，而且乘客方向也是上行，所以可以插入目标列表：

```text
TargetFloors: [10] -> [6, 10]
```

这样电梯会先在 6 楼停靠，再继续去 10 楼。

下行同理：

```text
CurrentFloor = 12
ScanDirection = down
TargetFloors = [2]
新增请求：8 楼下行
结果：TargetFloors = [8, 2]
```

#### 为什么 hall 请求还要看请求方向

Hall 请求是乘客在电梯外面按的“上”或“下”按钮。

如果电梯正在上行，它顺路接一个“上行 hall 请求”是合理的：乘客想上去，电梯也正在往上走。

但如果电梯正在上行，却在 6 楼顺路接一个“下行 hall 请求”，乘客真正想往下走。电梯把他接进去之后还会继续上行，这就不符合乘客意图。

所以当前规则是：

```text
运行中的电梯追加 hall 请求时：
楼层必须在当前扫描方向上
请求方向也必须和扫描方向一致
```

Cabin 请求是电梯内部楼层按钮，它不表示“我要上/下”，只表示“我要去某层”，所以 cabin 请求不需要和 hall 请求一样检查方向。

#### 空闲电梯如何接单

如果没有运行中的电梯能顺路接请求，就进入空闲电梯分配。

空闲电梯仍然保留 `ScanDirection`。这表示它下一次优先沿哪个方向找请求：

```text
优先选择位于 ScanDirection 方向上的请求
如果没有同方向请求，再选择反方向请求，并把 ScanDirection 调整过去
```

这比“永远选最近请求”更像 SCAN，因为 SCAN 的重点不是单次最近，而是维持一个扫描方向，减少反复掉头。

#### 为什么要排序 `TargetFloors`

顺路追加请求后，目标楼层不能简单 append 到末尾。

上行时应该从低到高服务：

```text
[10] + 6 -> [6, 10]
```

下行时应该从高到低服务：

```text
[2] + 8 -> [8, 2]
```

所以 `insertTargetFloor()` 会先去重，再根据 `ScanDirection` 对 `TargetFloors` 排序。

去重和删除 pending request 这种通用小操作放在 `internal/elevator/utils.go`，这样 `SCANScheduler.go` 可以更集中地表达调度算法本身。

#### 当前仍然保留的限制

当前模型仍然比较简单：

- `TargetFloors` 只保存楼层，不保存完整 `Request`。
- 没有开门等待时间。
- 没有并发 goroutine。
- 没有统计请求等待时间。

但这一步已经让 SCAN 从“只会给空闲电梯派单”升级成“运行中的电梯可以顺路追加请求”，更接近真实电梯调度。

本次新增的测试覆盖了：

- 上行途中把 6 楼插入 `[10]`，得到 `[6, 10]`。
- 下行途中把 8 楼插入 `[2]`，得到 `[8, 2]`。
- 没有同方向请求时，空闲电梯会反向接单。

### 2026-05-07：如果要实现更实际、更高质量的 SCAN，需要先改什么

这次“顺路请求可追加”的 SCAN 仍然偏快，适合作为中间版本，但还不是一个足够强的课程核心算法。

首先要明确一点：电梯调度里没有一个在所有情况下都绝对最优的算法。因为“最优”可能指不同目标：

```text
平均等待时间最短
最大等待时间最短
电梯总移动距离最短
乘客乘坐时间最短
电梯负载更均衡
避免某些楼层长期饥饿
```

这些目标有时会互相冲突。例如最近电梯优先可能降低当前请求等待时间，但可能破坏某部电梯的扫描方向，导致整体效率变差。SCAN/LOOK 牺牲一部分“单次最近”，换来更稳定、可解释、不会频繁掉头的行为。

所以本项目更合适的目标不是“数学上绝对最优”，而是：

```text
实现一个接近真实电梯集群控制的、可解释的、可测试的 SCAN/LOOK 调度算法。
```

#### 当前模型为什么不够

现在 `Elevator` 里用：

```go
TargetFloors []int
```

这只能表示“电梯接下来要停哪些楼层”，但丢失了很多重要信息：

```text
这个目标来自 hall 请求还是 cabin 请求？
hall 请求是上行还是下行？
这个请求什么时候创建？
这个请求是否已经分配给某部电梯？
这个目标楼层是否包含多个请求？
到达这一层时应该清除哪些请求？
```

例如：

```text
TargetFloors = [6]
```

无法区分：

```text
6 楼有人按了上行按钮
6 楼有人按了下行按钮
电梯内部有人按了 6 楼
上面几种情况同时存在
```

这会直接限制算法质量。真实调度不能只看楼层数字，还要看请求类型、方向、等待时间和是否已分配。

#### 建议的数据结构升级方向

第一步，给 `Request` 增加身份和时间信息。

可以考虑：

```go
type Request struct {
	ID        int64       `json:"id"`
	Floor     int         `json:"floor"`
	Direction Direction   `json:"direction"`
	Kind      RequestKind `json:"kind"`
	CreatedAt time.Time  `json:"createdAt"`
	AssignedElevatorID int `json:"assignedElevatorId"`
}
```

这里的重点是：

```text
ID
  用于唯一标识请求，方便测试、删除、调试。

CreatedAt
  用于计算等待时间，后续可以做 aging，避免请求一直等不到。

AssignedElevatorID
  用于区分“还没分配”和“已经分给某部电梯但尚未完成”。
```

不过 `AssignedElevatorID` 用 `0` 或 `-1` 表示未分配不够优雅。Go 里也可以先简单使用：

```go
AssignedElevatorID int `json:"assignedElevatorId"`
```

约定 `0` 表示未分配，因为电梯 ID 从 1 开始。

第二步，把 `TargetFloors []int` 升级成更有语义的停靠计划。

一种比较清晰的设计是：

```go
type StopPlan struct {
	Floor int `json:"floor"`

	PickUpUp   bool `json:"pickUpUp"`
	PickUpDown bool `json:"pickUpDown"`
	DropOff    bool `json:"dropOff"`

	RequestIDs []int64 `json:"requestIds"`
}
```

然后：

```go
type Elevator struct {
	ID              int
	CurrentFloor    int
	Direction       Direction
	ScanDirection   Direction
	DoorOpen        bool
	Stops           []StopPlan
	EmergencyStop   bool
}
```

这样同样是 6 楼停靠，可以表达得更清楚：

```text
6 楼停靠，接上行乘客
6 楼停靠，接下行乘客
6 楼停靠，有乘客下电梯
6 楼停靠，同时接上行乘客和放下乘客
```

这比 `TargetFloors []int` 更适合写高质量调度。

#### 为什么 `PendingRequests` 的语义也需要重新设计

这里容易误解：不是说一定不能保留 `PendingRequests` 这个字段名，而是说不能继续让它只表达“还没被调度器取走的请求”。

当前版本里的流程大概是：

```text
1. AddRequest 把请求加入 PendingRequests
2. Scheduler.Assign 从 PendingRequests 取出一个请求
3. 调度器把请求转成 Elevator.TargetFloors 里的一个楼层数字
4. 这个请求从 PendingRequests 中消失
```

这个设计在最小版本里能跑，但它有一个关键问题：

```text
请求一旦被分配给某部电梯，就从系统的请求列表里消失了。
```

它没有真的“完成”，只是“已经分配”。但当前模型把“已经分配”和“已经完成”混在一起了。

这会带来几个实际问题。

第一个问题：无法统计真实等待时间。

乘客按下按钮后，请求进入系统；电梯真正到达这一层并开门时，请求才算完成。如果请求被分配后就从 `PendingRequests` 删除，系统后续就很难知道：

```text
这个请求什么时候创建？
什么时候被分配？
什么时候真正被服务？
总共等待了多久？
```

没有这些信息，就很难实现“等待时间补偿”或“避免饥饿”的高质量调度。

第二个问题：无法准确取消或完成请求。

假设 6 楼有一个上行 hall 请求。调度器把它分给 1 号电梯后，如果只是把 `6` 放进 `TargetFloors`，后续电梯到了 6 楼时只能知道：

```text
我要在 6 楼停一下
```

但它不知道这个停靠对应的是：

```text
6 楼上行 hall 请求
6 楼下行 hall 请求
电梯内部 6 楼 cabin 请求
多个请求叠在一起
```

所以到站后也不知道应该清除哪个请求、保留哪个请求。

第三个问题：无法处理“同楼层不同方向”的情况。

例如同一时刻：

```text
6 楼有人要上行
6 楼也有人要下行
```

如果电梯上行经过 6 楼，它应该服务上行请求，但不一定应该清除下行请求。

如果系统只剩：

```text
TargetFloors = [6]
```

那么这两个请求已经被压缩成一个数字，方向信息丢失了。

第四个问题：调度器无法重新评估已经分配但尚未完成的请求。

真实系统里，一个请求分配给某部电梯后，不代表永远不能调整。比如：

```text
1 号电梯原本要去接 10 楼请求
但它中途加入了很多任务，预计到达变慢
2 号电梯突然空闲并且更近
```

如果系统还保留请求状态，就可以考虑重新分配。当前模型一旦从 `PendingRequests` 删除，就很难做这种优化。

所以因果关系是：

```text
想做更强的 SCAN/LOOK
  -> 需要知道请求是否 pending / assigned / done
  -> 需要知道请求的方向、时间、归属电梯
  -> 不能让请求在“刚分配”时就从系统视野里消失
  -> PendingRequests 的语义需要升级，或者替换成更完整的 Requests 列表
```

更推荐的设计是：系统保存所有尚未完成的请求，并给请求增加状态。

```go
type RequestStatus string

const (
	RequestPending  RequestStatus = "pending"
	RequestAssigned RequestStatus = "assigned"
	RequestDone     RequestStatus = "done"
)
```

然后 `Request` 可以变成：

```go
type Request struct {
	ID        int64         `json:"id"`
	Floor     int           `json:"floor"`
	Direction Direction     `json:"direction"`
	Kind      RequestKind   `json:"kind"`
	Status    RequestStatus `json:"status"`

	CreatedAt   time.Time `json:"createdAt"`
	AssignedAt  time.Time `json:"assignedAt"`
	CompletedAt time.Time `json:"completedAt"`

	AssignedElevatorID int `json:"assignedElevatorId"`
}
```

然后 `System` 不再只保存 `PendingRequests`，而是保存更完整的请求集合：

```go
Requests []Request
```

这样语义会更准确：

```text
RequestPending
  还没有分配给任何电梯

RequestAssigned
  已经分配给某部电梯，但乘客还没有被服务

RequestDone
  电梯已经到达对应楼层并完成服务
```

这种设计会让调度算法和报告都更有说服力。算法能解释“我为什么选择这部电梯”，报告也能展示“平均等待时间、最长等待时间、请求完成过程”等指标。

#### 是否完全不能保留 `PendingRequests`

可以保留，但要把它当成派生视图，而不是唯一事实来源。

例如：

```text
System.Requests
  保存所有未完成或历史请求，是事实来源

PendingRequests()
  从 Requests 中筛选 Status == pending 的请求
```

也就是说，`PendingRequests` 可以作为一个辅助概念存在，但核心状态应该是带状态的 `Requests`。

否则一旦请求被分配，它就会从系统里“消失”，这会直接限制后续的高质量调度。

#### 更实际的 SCAN/LOOK 应该怎么工作

严格 SCAN 会像扫描仪一样一直扫到最顶层或最底层再反向。实际电梯更常用 LOOK 思想：不一定扫到边界，只要当前方向上没有任何请求，就可以反向。

所以课程项目可以实现一个“集群 LOOK/SCAN”：

```text
1. 每部电梯维护自己的 ScanDirection。
2. 每部电梯维护 Stops，按扫描方向排序。
3. 新 hall 请求进入后，调度器给每部电梯计算一个 cost。
4. 如果某部电梯能顺路接，成本较低。
5. 如果某部电梯需要掉头，成本较高。
6. 如果某部电梯空闲，成本按距离计算。
7. 如果某请求等待时间很长，降低它的成本或提高优先级，避免饥饿。
8. 选择 cost 最低的电梯，把请求加入它的 Stops。
```

这种算法比单纯 SCAN 更适合 5 部电梯，因为它不是每个请求都机械地找一部空闲电梯，而是把“正在运行的电梯是否顺路”也考虑进去。

#### Cost 函数可以怎么设计

可以写一个可解释的评分函数：

```text
cost = 基础距离成本 + 掉头惩罚 + 已有任务惩罚 - 等待时间奖励
```

例如：

```text
顺路请求：
  cost = 电梯到请求楼层的距离

需要反向才能接：
  cost = 当前方向剩余路程 + 掉头惩罚 + 反向后到请求楼层的距离

空闲电梯：
  cost = 电梯到请求楼层的距离

请求等待很久：
  cost -= waitingSeconds / 2
```

这里不一定要追求公式复杂，关键是每一项都能解释：

```text
距离越近越好
顺路比掉头好
任务少的电梯更好
等待久的请求优先级提高
```

这在报告里也比较好写。

#### 到达某层时应该做什么

如果用 `StopPlan`，`stepElevator()` 到达某层后不应该只是删除一个 int，而应该：

```text
1. 找到当前楼层对应的 StopPlan
2. 打开门
3. 清除这一层已经完成的 cabin dropoff
4. 清除这一层、同方向的 hall pickup
5. 从 Stops 中删除这个 StopPlan
6. 如果当前方向上没有剩余 Stops，就根据剩余任务决定是否反向
```

这一步是“真实电梯模型”的关键。调度算法只是决定把请求交给谁，电梯运行逻辑要负责在到达楼层时正确完成请求。

#### 推荐的改造顺序

不要一次性把所有东西都改掉。比较稳妥的顺序是：

```text
1. 给 Request 增加 ID、CreatedAt、AssignedElevatorID。
2. 新增 StopPlan 结构体，但暂时保留 TargetFloors 或用 Stops 替代它。
3. 把 Elevator 的任务列表从 []int 升级为 []StopPlan。
4. 改 stepElevator，让它根据 Stops 停靠、开门、删除完成任务。
5. 改 AddRequest，让新请求带唯一 ID 和创建时间。
6. 改 SCAN/LOOK 调度器，用 cost 函数选择电梯。
7. 给核心场景写测试：顺路追加、反向、空闲接单、等待时间优先。
8. 稳定后再考虑并发模型。
```

#### 当前代码应该如何定位

当前 `SCANScheduler.go` 不应被看作最终版本，而应该看作：

```text
理解 SCAN 基本思想的过渡版本
```

它已经表达了几个重要概念：

```text
ScanDirection
顺路追加
hall 请求方向匹配
目标楼层排序
```

但如果目标是做课程里更有说服力的核心算法，下一步应该先改模型，而不是继续在 `TargetFloors []int` 上堆逻辑。因为数据结构不表达真实问题，算法就很难写得真实。

#### 当前结论

更高质量的路线是：

```text
先升级模型：
Request 有 ID / 时间 / 分配状态
Elevator 有 StopPlan 列表

再升级算法：
使用 LOOK/SCAN + cost 函数
顺路优先，掉头惩罚，等待时间补偿

最后再升级运行逻辑：
到站后根据 StopPlan 清除对应请求
```

这样实现出来的算法不一定是“所有场景数学最优”，但它会更接近真实电梯控制，也更容易在课程报告里解释清楚为什么合理。

### 2026-05-07：完成 6.5.2 请求状态模型重构

本次重构把请求从“临时 pending 队列里的元素”升级成了系统里的长期对象。

之前的核心状态是：

```go
PendingRequests []Request
```

调度器从里面取出一个请求后，请求就会从系统状态里消失。这样无法继续追踪它什么时候被分配、什么时候真正完成。

现在的核心状态改成：

```go
Requests []Request
```

每个请求自己带状态：

```go
const (
	RequestPending  RequestStatus = "pending"
	RequestAssigned RequestStatus = "assigned"
	RequestDone     RequestStatus = "done"
)
```

这表示：

```text
pending
  请求已经创建，但还没有分配给电梯。

assigned
  请求已经分配给某部电梯，但电梯还没到达对应楼层。

done
  电梯已经到达楼层并完成这个请求。
```

#### Request 新增字段

现在 `Request` 里新增了：

```go
ID int64
Status RequestStatus
CreatedTick int
AssignedTick int
CompletedTick int
AssignedElevatorID int
```

这些字段的作用是：

```text
ID
  唯一标识一个请求，方便调试、测试和后续 StopPlan 引用。

Status
  表示请求当前处于 pending / assigned / done 哪个阶段。

CreatedTick
  请求创建时的系统时间片。

AssignedTick
  请求被调度器分配给电梯时的系统时间片。

CompletedTick
  电梯真正到站并完成请求时的系统时间片。

AssignedElevatorID
  记录请求被哪部电梯接收。
```

#### 为什么继续使用离散 tick

这里仍然不用 `time.Time`，因为本项目现在是离散模拟系统。

真实时间关心的是“现在是几点几分几秒”。但电梯调度模拟更关心：

```text
第几个系统时间片创建请求
第几个系统时间片分配请求
第几个系统时间片完成请求
中间相差多少个时间片
```

因此：

```text
等待时间 = AssignedTick - CreatedTick
完成耗时 = CompletedTick - CreatedTick
```

这比真实时间更适合做可控测试。测试里连续调用 `Step()`，就能精确知道 tick 如何变化。

#### 调度器现在如何取 pending 请求

现在调度器不再读取 `PendingRequests` 字段。

如果需要找 pending 请求，会从 `Requests` 里筛选：

```go
request.Status == RequestPending
```

当前辅助函数包括：

```go
firstPendingRequestIndex(s)
hasPendingRequests(s)
requestIndicesByStatus(s, status)
```

这样 `Requests` 是事实来源，pending / assigned / done 都只是从事实来源里筛选出来的状态视图。

#### 临时的 `TargetRequestIDs`

当前还没有完成 6.5.4 的 `StopPlan`，所以电梯仍然保留：

```go
TargetFloors []int
```

为了能在到站时把对应请求标记为 `done`，临时增加了：

```go
TargetRequestIDs []int64
```

它和 `TargetFloors` 按下标对齐：

```text
TargetFloors     = [4, 8]
TargetRequestIDs = [1, 3]
```

含义是：

```text
先去 4 楼，完成 1 号请求
再去 8 楼，完成 3 号请求
```

这不是最终形态。后续 6.5.4 会用 `StopPlan` 替代这两个平行切片，因为平行切片容易维护出错，也无法表达同一楼层多个上下客动作。

#### 本阶段完成后的状态变化

创建请求时：

```text
Status = pending
CreatedTick = CurrentTick
AssignedTick = 0
CompletedTick = 0
AssignedElevatorID = 0
```

调度器分配请求时：

```text
Status = assigned
AssignedTick = CurrentTick
AssignedElevatorID = 电梯 ID
```

电梯到达目标楼层时：

```text
Status = done
CompletedTick = CurrentTick
```

这一步之后，系统已经可以观察请求从创建到完成的完整生命周期。后续就可以基于这些字段计算等待时间、完成时间，并为 cost 函数预留“等待时间补偿”能力。

### 2026-05-07：暂不做楼层人数和电梯负载建模

这次重新审视了 `docs/instructions-from-agent.md` 里的 6.5.3 和 6.5.4。结论是：当前核心模型不应该记录楼层等待人数，也不应该马上加入电梯负载人数。

原因是现实中的电梯调度系统通常只能知道：

```text
某层有人按了上行按钮
某层有人按了下行按钮
电梯内部有人按了某个目标楼层
```

也就是系统知道 request，但不知道：

```text
某层到底有几个人在等
这些人分别要去哪里
电梯到达后实际上会上来几个人
每个 request 背后有几个人
```

如果当前模型强行记录：

```go
Floor.UpWaitingCount
Floor.DownWaitingCount
Elevator.PassengerCount
```

就等于假设系统知道现实里不可直接观测的信息。这样会让核心调度算法变得复杂，而且不一定更真实。

#### 当前核心模型保留什么

当前核心模型保留：

```text
Request
  表示按钮请求。

Requests
  表示所有请求及其状态。

StopPlan
  后续用于表示电梯为什么要在某一层停靠。
```

因此 hall request 的语义是：

```text
某一层、某个方向的按钮被按下。
```

它不表示这一层有多少人。

到站时也只做：

```text
完成对应 request
更新 request 状态
打开门
移除停靠计划
```

不更新楼层人数，也不更新电梯人数。

#### 为什么 StopPlan 仍然要做

不做人数模型，不代表回到 `TargetFloors []int`。

`TargetFloors []int` 的问题仍然存在：它只知道要去某层，不知道为什么去这一层。

例如：

```text
6 楼上行 hall request
6 楼下行 hall request
电梯内部 6 楼 cabin request
```

如果都压缩成：

```go
TargetFloors = []int{6}
```

系统就无法区分这些请求。

所以 `StopPlan` 仍然是核心重构：

```text
StopPlan 负责表达“为什么在这一层停”
Floor / PassengerCount 暂时不做
```

#### 后续扩展

楼层人数、电梯负载、上下客耗时可以作为最后的扩展 TODO。

这些扩展可以用于：

```text
做更丰富的模拟
展示算法评价指标
让系统演示更真实
```

但它们不应该阻塞当前主线：

```text
Request 状态模型
StopPlan
调度算法
并发模型
API 和前端切换
```

关于系统初始化参数，当前先不使用 `SystemConfig`。楼层数、电梯数、跨楼层耗时、开门基础耗时等参数直接作为 `NewSystem(...)` 的参数传入，并保存在 `System` 字段里。这样和当前的 `FloorCount` 设计保持一致。后续如果前端需要动态修改，再设计 API 扩展。

### 2026-05-07：确定 `Step()` 和 tick 的最终语义

这次重新整理了时间片模型。目标是让系统看起来像现实世界中的离散模拟：

```text
前端定时调用一次 /api/step
后端系统进入下一个 tick
调度器在这个 tick 做即时决策
每部电梯在这个 tick 执行一个动作单位
前端再读取 /api/state 显示结果
```

因此当前规则是：

```text
每调用一次 Step()，CurrentTick 先 +1。
CurrentTick 表示“正在处理的 tick 编号”。
调度分配发生在这个 tick 内，但调度本身不额外消耗 tick。
电梯移动、开门停靠等物理动作消耗 tick。
```

`Step()` 的顺序是：

```text
1. CurrentTick += 1
2. scheduler.Assign(s) 尝试分配 pending request
3. 每部电梯执行一个行动单位
4. 返回当前状态
```

这样时间戳的含义更直接：

```text
CreatedTick
  请求被创建时，系统已经处理到哪个 tick。

AssignedTick
  请求在哪个 tick 被调度器分配给电梯。

CompletedTick
  请求在哪个 tick 被电梯到站开门完成。
```

例如系统启动时：

```text
CurrentTick = 0
```

此时创建一个请求：

```text
CreatedTick = 0
```

下一次前端触发 `Step()`：

```text
CurrentTick = 1
调度器分配请求
AssignedTick = 1
电梯执行第 1 个行动单位
```

#### `TicksPerFloor` 现在如何生效

之前虽然 `System` 有：

```go
TicksPerFloor int
```

但电梯每次 `Step()` 都直接移动一层，所以这个字段没有真正参与模拟。

现在 `Elevator` 增加了：

```go
MoveRemainingTicks int
```

它表示当前跨越相邻两层还剩多少 tick。

例如：

```text
TicksPerFloor = 3
电梯从 1 楼去 2 楼
```

那么：

```text
Step 1:
  MoveRemainingTicks 从 3 变成 2
  电梯仍在 1 楼

Step 2:
  MoveRemainingTicks 从 2 变成 1
  电梯仍在 1 楼

Step 3:
  MoveRemainingTicks 从 1 变成 0
  电梯到达 2 楼
```

这样 `TicksPerFloor` 才真正表达了“跨越一层需要几个时间片”。

#### 开门时间如何生效

现在 `Elevator` 还增加了：

```go
DoorRemainingTicks int
```

当电梯到达目标楼层后，会：

```text
1. 打开门
2. 完成对应 request
3. 设置 DoorRemainingTicks = System.DoorBaseTicks
```

之后每个 `Step()` 会让 `DoorRemainingTicks` 减 1。减到 0 后，门关闭。

当前还没有人数模型，所以 `TickPerPassenger` 暂时还不会进入计算。它会等后续扩展“电梯负载 / 上下客耗时模型”时再使用。

#### 这次重构后的结论

现在时间片语义更清楚：

```text
Step 触发 tick 前进
调度发生在 tick 内，不额外计时
移动按 TicksPerFloor 消耗多个 tick
开门停靠按 DoorBaseTicks 消耗多个 tick
CurrentTick 始终表示已经进入并正在处理的 tick
```

这也让后续调度算法可以更准确地解释：

```text
请求什么时候创建
请求什么时候被分配
电梯花了多少 tick 才到达
请求什么时候真正完成
```

### 2026-05-07：用 `StopPlan` 替代 `TargetFloors []int`

本次完成了 6.5.4 的核心重构：不再用两个平行切片表示电梯任务。

之前的临时模型是：

```go
TargetFloors []int
TargetRequestIDs []int64
```

它的问题是：

```text
TargetFloors 只知道“去哪一层”
TargetRequestIDs 只知道“完成哪些请求”
两者必须依靠下标对齐
同一楼层不同方向请求很容易被错误合并
```

例如：

```text
6 楼 hall up
6 楼 hall down
6 楼 cabin target
```

如果只保存：

```go
TargetFloors = []int{6}
```

系统就不知道电梯在 6 楼停靠的原因是什么。

#### 新结构

现在新增了：

```go
type StopReason string

const (
	StopReasonHallUp   StopReason = "hall_up"
	StopReasonHallDown StopReason = "hall_down"
	StopReasonCabin    StopReason = "cabin"
)

type StopPlan struct {
	Floor      int        `json:"floor"`
	Reason     StopReason `json:"reason"`
	Direction  Direction  `json:"direction"`
	RequestIDs []int64    `json:"requestIds"`
}
```

`Elevator` 里改成：

```go
Stops []StopPlan `json:"stops"`
```

所以现在一部电梯的任务不再是“目标楼层列表”，而是“停靠计划列表”。

#### StopPlan 的语义

一个 `StopPlan` 表示：

```text
电梯需要在 Floor 这一层停靠
停靠原因是 Reason
这个停靠会完成 RequestIDs 里的请求
```

例如：

```text
StopPlan{
  Floor: 6,
  Reason: hall_up,
  Direction: up,
  RequestIDs: [1, 3],
}
```

表示：

```text
电梯要在 6 楼停靠，用于完成 1 号和 3 号上行 hall 请求。
```

如果同一层还有下行请求，则会是另一个 `StopPlan`：

```text
StopPlan{
  Floor: 6,
  Reason: hall_down,
  Direction: down,
  RequestIDs: [2],
}
```

这样同楼层不同方向不会被错误合并。

#### 合并规则

当前合并规则在 `addStopPlan()` 里：

```text
Floor 相同
Reason 相同
Direction 相同
```

才会合并为同一个停靠计划，并把 request ID 追加进去。

如果楼层相同但 reason 不同，例如 hall up 和 hall down，就不会合并。

#### 调度器如何使用

普通调度器分配请求时，会调用：

```go
s.assignRequestToElevator(requestIndex, elevatorIndex)
```

这个函数会：

```text
1. 把请求状态改为 assigned
2. 记录 AssignedTick
3. 记录 AssignedElevatorID
4. 根据 request 生成 StopPlan
5. 把 StopPlan 加入 elevator.Stops
```

SCAN 调度器也改成插入 `StopPlan`，并继续按扫描方向排序：

```text
上行：楼层从低到高
下行：楼层从高到低
```

#### 电梯到站如何完成请求

`stepElevator()` 现在读取：

```go
nextStop := e.Stops[0]
```

而不是：

```go
targetFloor := e.TargetFloors[0]
```

当电梯到达 `nextStop.Floor` 后：

```text
1. 开门
2. 遍历 nextStop.RequestIDs
3. 把这些请求标记为 done
4. 从 Stops 中移除这个 StopPlan
```

#### 测试覆盖

这次补充和调整了测试：

```text
同一楼层 hall up 和 hall down 不会被错误合并
SCAN 上行顺路插入后，Stops 顺序是 [6, 10]
SCAN 下行顺路插入后，Stops 顺序是 [8, 2]
到站后 Stops 会被移除，请求状态会变成 done
```

#### 当前限制

`StopPlan` 已经解决了“为什么停靠”的表达问题，但它仍然是中间阶段：

```text
还没有并发模型
还没有最终 LOOK / SCAN cost 函数
还没有前端展示 stops
还没有电梯人数和负载模型
```

其中人数和负载已经被移动到后续扩展 TODO，不进入当前核心路线。

### 2026-05-07：关于 `Requests` 是否应该继续使用切片

这次重新审视了当前请求存储结构：

```go
Requests []Request
nextRequestID int64
```

这里有两个容易混淆的问题。

#### 在当前实现里，`Request.ID` 和 `Requests` 下标是否等价

要分两层看。

从**当前实现的数值结果**看，它们几乎是等价的，但有一个 `+1` 偏移。

当前代码里，请求 ID 来自 `System.nextRequestID`：

```go
ID: s.nextRequestID
s.nextRequestID++
```

同时，当前请求只会被追加到切片末尾：

```go
s.Requests = append(s.Requests, request)
```

完成请求时，当前代码只是把请求状态改成 `done`，并不会从 `Requests` 里删除它。

所以在当前代码满足这几个条件时：

```text
只 append
不 delete
不排序
nextRequestID 从 1 开始，每次加 1
```

那么第一个请求会是：

```text
Request.ID = 1
Requests[0]
```

第二个请求会是：

```text
Request.ID = 2
Requests[1]
```

也就是：

```text
Request.ID == Requests 下标 + 1
```

从这个角度说，你的观察是对的：**在目前实现语境下，ID 和下标存在稳定对应关系**。

但从**设计语义**上看，它们不应该被认为是同一个东西。

也就是说：

```text
Request.ID     是系统分配的稳定编号
Requests[i]    是当前切片里的第 i 个元素
```

当前只是因为 `Requests` 从不删除，所以二者看起来绑定了。

原因是：下标只表示“当前放在切片里的位置”。如果以后删除某个请求，或者对请求重新排序，切片下标就可能变化；但请求 ID 应该在请求生命周期内保持不变，因为 `StopPlan.RequestIDs`、日志、测试、前端展示都可能通过这个 ID 引用同一个请求。

所以更合理的原则是：

```text
Request.ID 是身份标识
切片下标只是某种存储方式的内部细节
```

#### 完成的请求是否应该一直留在 `Requests` 里

当前实现会把完成请求标记为：

```go
Status: RequestDone
```

但请求对象仍然留在 `Requests []Request` 里。

这个设计在早期有一个好处：容易观察系统历史状态。比如前端或测试可以看到：

```text
这个请求什么时候创建
什么时候分配
什么时候完成
总等待了多久
```

但是它的问题也很明显：如果系统长期运行，所有已经完成的请求都会一直留在内存里。后续查找某个请求时也需要遍历整个切片：

```go
for i := range s.Requests {
    if s.Requests[i].ID == requestID {
        ...
    }
}
```

这对课程 demo 来说暂时不会炸，但从系统设计上看，它把两个职责混在了一起：

```text
运行态请求表：系统现在还需要处理哪些请求
历史记录：系统过去处理过哪些请求，用于统计和展示
```

这两个职责最好分开。

#### 更合理的方向：用 map 存运行态请求

Go 里没有叫 `unordered_map` 的标准类型；对应概念是 `map`。

可以把运行态请求改成：

```go
Requests map[int64]*Request
```

含义是：

```text
key   = Request.ID
value = 指向 Request 的指针
```

这样做之后，请求 ID 就成为真正的索引：

```go
request := s.Requests[requestID]
```

而不需要在切片里线性查找。

请求完成后，也可以从运行态 map 中删除：

```go
delete(s.Requests, requestID)
```

这样 `Requests` 就只保存“当前系统还关心的活跃请求”，不会因为已经完成的历史请求无限增长。

#### 但是历史统计不能直接丢

需要注意：删除完成请求不代表系统不需要统计数据。

后续如果要做等待时间补偿、平均等待时间、最大等待时间、调度算法对比，就不能把完成请求的信息彻底丢掉。更合理的做法是把运行态请求和统计结果分开：

```text
Requests map[int64]*Request
  保存 pending / assigned 等仍在系统中的请求

RequestStats 或 CompletedRequestHistory
  保存已经完成请求的等待时间、完成时间、总数等统计数据
```

当前下一步可以先做前半部分：

```text
把 Requests 改成 map[int64]*Request
完成请求后从运行态 Requests 删除
暂时只预留后续统计结构，不急着做完整历史数据库
```

这样模型语义会更清楚：

```text
StopPlan.RequestIDs 引用请求 ID
System.Requests 负责通过 ID 找到活跃请求
完成后从 System.Requests 删除
后续统计另建字段保存
```

这个方向已经写入 `docs/instructions-from-agent.md` 的 6.5.5，作为下一步计划。

### 2026-05-07：历史统计应该放在哪里

历史统计不应该长期堆在 `System.Requests` 这种核心运行态结构里。

可以把系统里的数据分成两类：

```text
运行控制状态
  系统为了继续运行必须知道的数据
  例如：当前活跃请求、每部电梯的位置、Stops、门状态、调度器状态

历史观测数据
  系统为了分析、展示、审计、写报告而记录的数据
  例如：某个请求创建时间、分配时间、完成时间、等待时长、使用了哪个调度算法
```

`System` 作为电梯运行核心，应该优先保存第一类数据。否则运行时间越长，核心结构越像一个日志数据库，调度器每次查状态也会越来越重。

#### 历史请求更像日志

一个已经完成的请求，对“下一步电梯怎么移动”通常没有直接作用。

它更像一条事件记录：

```text
tick=10 request=42 created floor=6 direction=up
tick=12 request=42 assigned elevator=2
tick=30 request=42 completed elevator=2
```

这些信息当然很有价值，但价值主要在于：

```text
计算平均等待时间
比较不同调度算法
生成课程报告里的实验数据
调试为什么某个请求等了很久
后续如果做可视化，可以画等待时间曲线
```

这些用途更接近日志、监控或统计，而不是电梯核心控制状态。

#### 长期运行时，数据库或日志文件更合适

如果这是一个长期运行的真实系统，历史记录通常不应该只存在内存里，而应该写到外部持久化位置，例如：

```text
数据库
日志文件
事件流
监控系统
```

数据库适合查询，比如：

```text
过去 1 小时平均等待时间是多少
哪部电梯完成请求最多
SCAN 和 FCFS 在同一批请求下哪个等待时间更短
某个请求 ID 的完整生命周期是什么
```

这种能力不适合塞进 `System` 结构体本身。`System` 更应该回答：

```text
现在有哪些活跃请求
每部电梯现在在哪里
下一步该怎么调度
```

#### 当前项目可以怎么处理

本项目现在还没有数据库，也不急着引入数据库。

比较合理的阶段设计是：

```text
第一步：
  System.Requests 改成 map[int64]*Request
  只保存 pending / assigned 等活跃请求
  请求完成后从 map 删除

第二步：
  在完成请求时生成一条轻量的完成记录或统计增量
  例如 CompletedRequestCount、TotalWaitTicks、MaxWaitTicks

第三步：
  如果后续需要实验报告数据，再考虑把事件追加写入文件
  例如 logs/requests.jsonl

第四步：
  如果项目进一步工程化，再考虑数据库
```

也就是说，方向上你说的是对的：历史完成请求更适合放在核心结构之外，甚至可以持久化到数据库。

只是以当前课程项目的复杂度来说，马上接数据库会把学习重点带偏。更稳的边界是：

```text
核心运行态：System
短期统计：SystemStats 或 RequestStats
长期历史：日志文件 / 数据库
```

这样既不丢统计能力，也不会让 `System.Requests` 变成越来越大的历史仓库。

## 2026-05-08：请求运行态存储重构（map + 临时历史记录）

本阶段目标：把 `System.Requests` 从 `[]Request` 升级为 `map[int64]*Request`。当时为了先完成运行态和历史的拆分，临时把完成请求移入 `RequestHistory`。

注意：`RequestHistory` 只是 6.5.5 的过渡结构。后续 6.5.7 已经把它删除，改为写入 SQLite。

### 为什么换 map

之前 `Requests` 是 `[]Request`，查找某个请求需要 O(n) 遍历。`completeRequest`、`assignRequestToElevator`、调度器的 `Assign` 都要线性搜索。

换成 `map[int64]*Request` 后，通过请求 ID 直接 O(1) 查找。

### 运行态和历史分离

```text
运行态 Requests（map[int64]*Request）：
  只有 pending 和 assigned 状态

临时历史 RequestHistory（[]*Request）：
  请求完成后从 Requests 删除，append 到这里
  6.5.7 中已经被 SQLite 替代
```

### 辅助函数的类型变化

```go
// 旧：返回切片下标 int
func firstPendingRequestIndex(s *System) int
func requestIndicesByStatus(s *System, status RequestStatus) []int

// 新：返回 map key int64
func firstPendingRequestID(s *System) int64
func requestIDsByStatus(s *System, status RequestStatus) []int64
```

空值检查也从 `-1` 改为 `0`（int64 零值）。

### Go 语法注意：map 的 range

```go
// 切片遍历：拿到的是值拷贝
for i, request := range s.Requests {
    // i 是 int（下标）
    // request 是 Request（值拷贝）
}

// map 遍历：value 本身就是指针
for id, request := range s.Requests {
    // id 是 int64（key）
    // request 是 *Request（指针，无需再取地址）
}
```

### 本次验证

```bash
go build ./...
go test ./...
```

全部通过（9 个测试）。

## 2026-05-08：完成请求历史 SQLite 持久化

本阶段完成 6.5.7：运行态 `Requests map[int64]*Request` 继续只保存活跃请求；请求完成时，不再进入内存里的 `RequestHistory`，而是写入 SQLite 数据库。

### Go 依赖

项目新增 SQLite Go 驱动：

```text
github.com/mattn/go-sqlite3
```

它是 `database/sql` 的 SQLite driver。代码里真正使用的是 Go 标准库的 `database/sql`，SQLite 驱动通过空白导入注册：

```go
import _ "github.com/mattn/go-sqlite3"
```

这里的 `_` 表示：导入这个包是为了执行它的初始化逻辑，而不是直接调用包里的函数。

### 新增 `RequestStore`

新增文件：

```text
internal/elevator/request_store.go
```

它封装数据库逻辑，避免 SQL 语句散落在 `system.go` 里面。

主要函数：

```go
OpenRequestStore(databasePath string)
SaveCompletedRequest(request Request)
CompletedRequestCount()
CompletedRequestByID(requestID int64)
MaxCompletedRequestID()
Close()
```

`System` 内部增加了一个不导出的字段：

```go
requestStore *RequestStore `json:"-"`
```

字段名小写，说明它只属于后端内部；`json:"-"` 表示它不会出现在 `GET /api/state` 的 JSON 里。

### SQLite 表结构

数据库表名：

```text
completed_requests
```

表字段和 `Request` 结构体一一对应：

```text
id
floor
direction
kind
status
created_tick
assigned_tick
completed_tick
assigned_elevator_id
```

这满足当前文档里的要求：表的属性和一个 `Request` 保持一致。

### 完成请求时发生什么

现在 `completeRequest()` 的流程是：

```text
1. 通过 requestID 从运行态 Requests map 中找到请求
2. 复制一份 completedRequest
3. 设置 Status = done
4. 设置 CompletedTick
5. 调用 requestStore.SaveCompletedRequest(...) 写入 SQLite
6. 写入成功后，更新原请求对象
7. 从运行态 Requests map 中 delete
```

注意顺序：先写数据库，再从运行态 map 删除。

这样做是为了避免数据库写入失败时，内存中的活跃请求已经被删掉，导致历史记录和运行状态同时丢失。

### 删除 `RequestHistory`

之前 6.5.5 里临时使用过：

```go
RequestHistory []*Request
```

现在它已经删除。

原因是：既然 6.5.7 明确要求上数据库，那么历史完成请求就不应该再同时保存在内存历史切片里。否则会出现两个历史来源：

```text
RequestHistory 内存切片
SQLite completed_requests 表
```

两个来源并存会增加同步问题。当前设计里，历史完成请求的事实来源是 SQLite。

### 系统初始化和数据库文件

测试里默认使用内存 SQLite：

```go
NewSystem(...)
```

内部会使用：

```text
:memory:
```

正式后端启动时使用文件数据库：

```go
NewSystemWithDatabase(..., "data/requests.db")
```

也就是说，运行：

```bash
go run ./cmd/server
```

之后，请求完成记录会写入：

```text
data/requests.db
```

`OpenRequestStore()` 会自动创建数据库目录。

### 为什么要读取最大 ID

SQLite 表中 `id` 是主键。

如果服务器重启后 `nextRequestID` 又从 1 开始，那么新的完成请求写入数据库时可能和旧记录冲突。

所以 `NewSystemWithDatabase()` 初始化时会调用：

```go
MaxCompletedRequestID()
```

然后让：

```text
nextRequestID = maxCompletedRequestID + 1
```

这样重启后，请求 ID 会继续向后增长。

### 本次验证

已运行：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go mod tidy
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

结果：

```text
cmd/server: no test files
internal/api: no test files
internal/elevator: ok
```

因为当前沙箱里的默认 Go build cache 目录只读，所以临时把 `GOCACHE` 指向了 `/tmp/os_sp26_proj1-go-build`。

### 2026-05-09 关于数据库持久化的一个思考

其实我觉得现在有两个设计不太合理：

1. `MaxCompletedRequestID` 的设计不太合理。其实我的理想情况是每次的 `RequestID` 都直接从 0 开始计数，每次不同的运行会新建一个表。但是这里又有新的问题：什么是“一次运行”？我发现我不是很说得清楚，因为我对最终的界面的设想其实还不是很完善。我觉得这个先列为 TODO，我之后会完善的。Agent 阅读到这里的话也别对这一段文字别做任何修改。
2. `NewSystemWithDatabase` 的设计我也觉得很烂...因为这里相当于是 `NewSystem` 返回一个 `NewSystemWithDataSystem` 的闭包，我姑且认为这样的设计的缘由是为了向前兼容，防止重构太多东西，但是有点又臭又长了，我不是很喜欢。 Update: 已修改，全部重构为 `NewSystem`

## 2026-05-09：阅读 `internal/elevator/request_store.go`

这一节专门解释 `internal/elevator/request_store.go`。这是项目里第一次比较正式地接触后端数据库代码。

### 这个文件负责什么

`request_store.go` 的职责是：

```text
把已经完成的 Request 写入 SQLite
从 SQLite 读取已经完成的 Request
初始化数据库表
关闭数据库连接
```

这个文件本质上是一个持久化层：

```text
System 负责运行逻辑
RequestStore 负责数据库读写
SQLite 负责把历史请求保存到磁盘
```

### import 部分

```go
import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)
```

逐个看：

```text
database/sql
  Go 标准库提供的数据库统一接口。
  它不直接实现 SQLite、MySQL 或 PostgreSQL，而是定义一套通用操作方式。

fmt
  用来创建带上下文的 error。

os
  用来创建数据库目录。

path/filepath
  用来处理文件路径，例如获取 data/requests.db 的目录 data。

github.com/mattn/go-sqlite3
  SQLite 驱动。
```

这里最特殊的是：

```go
_ "github.com/mattn/go-sqlite3"
```

前面的 `_` 叫空白导入。意思是：

```text
我不在代码里直接调用这个包的函数，
但我要让这个包的 init() 执行，
从而把 sqlite3 这个数据库驱动注册给 database/sql。
```

如果没有这行，后面这句会找不到 SQLite 驱动：

```go
sql.Open("sqlite3", databasePath)
```

### `RequestStore` 结构体

```go
type RequestStore struct {
	db *sql.DB
}
```

`RequestStore` 是我们自己定义的一层包装。

`*sql.DB` 是 Go 标准库里的数据库连接池对象。虽然名字叫 `DB`，但它不是一个单独的数据库连接，而是一个可以管理连接的对象。

为什么不直接在 `System` 里放 `*sql.DB`？

因为直接放 `*sql.DB` 会让 `system.go` 里到处出现 SQL 语句，核心运行逻辑会变得很乱。

现在的分工是：

```text
System:
  只知道 requestStore.SaveCompletedRequest(...)

RequestStore:
  知道 SQL 表名、字段名、INSERT / SELECT 语句
```

这就是一层很薄的封装。

### `OpenRequestStore`

```go
func OpenRequestStore(databasePath string) (*RequestStore, error)
```

这个函数负责打开数据库。

流程是：

```text
1. 检查 databasePath 不能为空
2. 如果是文件数据库，确保目录存在
3. 调用 sql.Open("sqlite3", databasePath)
4. 创建 RequestStore
5. 调用 initSchema() 确保表存在
6. 返回 store
```

这里的：

```go
sql.Open("sqlite3", databasePath)
```

意思是：

```text
使用名为 sqlite3 的驱动
打开 databasePath 指向的 SQLite 数据库
```

如果 `databasePath` 是：

```text
data/requests.db
```

那么 SQLite 会把数据保存到这个文件。

如果 `databasePath` 是：

```text
:memory:
```

那么 SQLite 会创建一个内存数据库，程序结束后数据消失。测试里常用这个，因为不会污染项目目录。

### 为什么要 `initSchema`

```go
if err := store.initSchema(); err != nil {
	db.Close()
	return nil, err
}
```

打开数据库文件不等于里面已经有表。

所以启动时要确保表存在。`initSchema()` 里用了：

```sql
CREATE TABLE IF NOT EXISTS completed_requests (...)
```

这句 SQL 的意思是：

```text
如果 completed_requests 表不存在，就创建它；
如果已经存在，就什么都不做。
```

这样服务可以重复启动，不会因为表已经存在而报错。

如果建表失败，代码会：

```go
db.Close()
return nil, err
```

这是为了避免数据库已经打开，但初始化失败时连接泄漏。

### `Close`

```go
func (s *RequestStore) Close() error
```

后端程序打开数据库后，理论上应该在程序结束时关闭它。

这里先判断：

```go
if s == nil || s.db == nil {
	return nil
}
```

这是防御式写法。意思是：如果 `RequestStore` 本身不存在，或者里面的 `db` 还没初始化，那关闭操作直接视为成功。

真正关闭数据库的是：

```go
return s.db.Close()
```

在 `cmd/server/main.go` 里通常会配合：

```go
defer system.Close()
```

这样 `main()` 结束时会释放数据库资源。

### `SaveCompletedRequest`

```go
func (s *RequestStore) SaveCompletedRequest(request Request) error
```

这个函数负责把一个完成的请求写入数据库。

核心 SQL 是：

```sql
INSERT INTO completed_requests (
	id,
	floor,
	direction,
	kind,
	status,
	created_tick,
	assigned_tick,
	completed_tick,
	assigned_elevator_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
```

`INSERT INTO` 表示插入一行数据。

字段列表：

```text
id
floor
direction
kind
status
created_tick
assigned_tick
completed_tick
assigned_elevator_id
```

和 `Request` 的字段基本一一对应。

`VALUES (?, ?, ...)` 里面的 `?` 是占位符。真正的值在后面传入：

```go
request.ID,
request.Floor,
request.Direction,
request.Kind,
request.Status,
request.CreatedTick,
request.AssignedTick,
request.CompletedTick,
request.AssignedElevatorID,
```

这样写比自己拼字符串安全，也更标准。

不要写成：

```go
"INSERT ... VALUES (" + someValue + ")"
```

因为手动拼 SQL 容易出错，也可能引入 SQL 注入问题。即使当前项目数据主要来自我们自己的类型，仍然应该习惯使用占位符。

这里使用的是：

```go
s.db.Exec(...)
```

`Exec` 执行不返回结果集的 SQL，比如 `INSERT` `DELETE` `CREATE TABLE` 等。

### `CompletedRequestCount`

```go
func (s *RequestStore) CompletedRequestCount() (int, error)
```

这个函数返回数据库里已经完成的请求数量。

SQL 是：

```sql
SELECT COUNT(*) FROM completed_requests
```

这里使用：

```go
s.db.QueryRow(...).Scan(&count)
```

`QueryRow` 适合查询只返回一行的 SQL。

`Scan(&count)` 的意思是：

```text
把数据库查询结果写入 Go 变量 count
```

注意这里传的是：

```go
&count
```

因为 `Scan` 需要修改这个变量，所以要传指针。

### `MaxCompletedRequestID`

```go
func (s *RequestStore) MaxCompletedRequestID() (int64, error)
```

这个函数读取数据库里最大的请求 ID。

SQL 是：

```sql
SELECT MAX(id) FROM completed_requests
```

用途是：系统重启后，新的请求 ID 不要又从 1 开始。

如果数据库里已经有：

```text
id = 1, 2, 3
```

那么最大 ID 是 3，系统下一个请求应该从 4 开始。

这里用了：

```go
var maxID sql.NullInt64
```

为什么不用普通 `int64`？

因为如果表是空的：

```sql
SELECT MAX(id) FROM completed_requests
```

返回的是 SQL 里的 `NULL`，不是 0。

`sql.NullInt64` 可以表达两种状态：

```text
Valid == true
  数据库真的返回了一个整数

Valid == false
  数据库返回的是 NULL
```

所以代码里有：

```go
if !maxID.Valid {
	return 0, nil
}
return maxID.Int64, nil
```

意思是：如果表里还没有任何历史请求，就认为最大 ID 是 0。

### `CompletedRequestByID`

```go
func (s *RequestStore) CompletedRequestByID(requestID int64) (*Request, error)
```

这个函数按 ID 查询一条已经完成的请求。

SQL 是：

```sql
SELECT
	id,
	floor,
	direction,
	kind,
	status,
	created_tick,
	assigned_tick,
	completed_tick,
	assigned_elevator_id
FROM completed_requests
WHERE id = ?
```

`WHERE id = ?` 表示只查指定 ID 的那一行。

后面的：

```go
`, requestID).Scan(
	&request.ID,
	&request.Floor,
	&request.Direction,
	&request.Kind,
	&request.Status,
	&request.CreatedTick,
	&request.AssignedTick,
	&request.CompletedTick,
	&request.AssignedElevatorID,
)
```

表示：

```text
把查询出来的每一列，按顺序写入 request 的对应字段。
```

这里字段顺序很重要。

SQL 里 SELECT 的顺序是：

```text
id, floor, direction, kind, ...
```

`Scan` 里的目标也必须按这个顺序写：

```text
&request.ID, &request.Floor, &request.Direction, &request.Kind, ...
```

如果顺序写错，就会把数据库里的列读到错误字段里。

### `initSchema`

```go
func (s *RequestStore) initSchema() error
```

这个函数负责建表。

SQL：

```sql
CREATE TABLE IF NOT EXISTS completed_requests (
	id INTEGER PRIMARY KEY,
	floor INTEGER NOT NULL,
	direction TEXT NOT NULL,
	kind TEXT NOT NULL,
	status TEXT NOT NULL,
	created_tick INTEGER NOT NULL,
	assigned_tick INTEGER NOT NULL,
	completed_tick INTEGER NOT NULL,
	assigned_elevator_id INTEGER NOT NULL
)
```

逐个看：

```text
id INTEGER PRIMARY KEY
  id 是主键。主键用于唯一标识一行记录。

floor INTEGER NOT NULL
  floor 是整数，并且不能为空。

direction TEXT NOT NULL
  direction 是文本，例如 up/down/idle。

kind TEXT NOT NULL
  kind 是文本，例如 hall/cabin。

status TEXT NOT NULL
  status 是文本，例如 done。

created_tick / assigned_tick / completed_tick
  都是整数 tick。

assigned_elevator_id INTEGER NOT NULL
  记录这个请求被哪部电梯完成。
```

`NOT NULL` 表示这一列不允许为空。

这对当前项目很合适，因为一个完整的完成请求应该总能提供这些字段。

### `ensureDatabaseDirectory`

```go
func ensureDatabaseDirectory(databasePath string) error
```

这个函数负责确保数据库文件所在目录存在。

如果路径是：

```text
data/requests.db
```

那么：

```go
filepath.Dir(databasePath)
```

会得到：

```text
data
```

然后：

```go
os.MkdirAll(dir, 0755)
```

会创建这个目录。

`MkdirAll` 的特点是：

```text
目录不存在就创建
目录已经存在也不会报错
可以一次创建多级目录
```

为什么要特殊处理：

```go
if databasePath == ":memory:" {
	return nil
}
```

因为 `:memory:` 不是文件路径，而是 SQLite 的特殊写法，表示内存数据库。它没有目录，也不应该创建目录。

### 这个文件的整体调用链

后端启动时：

```text
main.go
  -> NewSystem(...)
    -> OpenRequestStore(...)
      -> initSchema()
```

请求完成时：

```text
Step()
  -> stepElevator()
    -> completeRequest()
      -> requestStore.SaveCompletedRequest()
      -> delete(System.Requests, requestID)
```

测试读取数据库时：

```text
system.requestStore.CompletedRequestCount()
system.requestStore.CompletedRequestByID(...)
```

### 后端数据库代码的基本模式

这个文件体现了后端写数据库时的一个常见模式：

```text
1. 定义一个 Store / Repository 类型
2. Store 内部持有数据库连接
3. 初始化时建表或检查 schema
4. 对外暴露业务语义函数，而不是暴露 SQL
5. 调用方只调用 Save / Find / Count 这类函数
```

例如这里不是让 `system.go` 写：

```go
db.Exec("INSERT INTO completed_requests ...")
```

而是写：

```go
s.requestStore.SaveCompletedRequest(completedRequest)
```

这样 `system.go` 读起来仍然是在表达业务逻辑：

```text
完成请求
保存历史
从运行态请求表删除
```

而不是被 SQL 细节打断。

## 2026-05-09：重构 `POST /api/request`

这次把 `internal/api/handler.go` 里的 `POST /api/request` 改成了更符合当前 `Request` 模型的版本。

### 客户端现在应该提交什么

前端或 curl 只需要提交客户端真正知道的信息：

```json
{
  "floor": 4,
  "direction": "up",
  "kind": "hall"
}
```

也就是：

```text
floor      请求楼层
direction  hall 请求的方向，up/down；cabin 请求用 idle
kind       hall 或 cabin
```

客户端不应该提交：

```text
id
status
createdTick
assignedTick
completedTick
assignedElevatorId
```

这些字段都属于后端运行状态，应该由 `System.AddRequest()` 创建。

### 为什么不能让客户端传完整 `Request`

之前 handler 直接把请求体解码成：

```go
var request elevator.Request
```

这会让 API 语义变得不清楚：客户端看起来可以自己指定 `ID`、`Status`、tick 字段。

但当前模型里：

```text
ID                由 nextRequestID 生成
Status            新请求默认是 pending
CreatedTick        使用当前 System.CurrentTick
AssignedTick       调度时填写
CompletedTick      电梯到站完成时填写
AssignedElevatorID 调度时填写
```

所以 handler 现在定义了专门的请求体结构：

```go
type createRequestPayload struct {
	Floor     int                 `json:"floor"`
	Direction elevator.Direction `json:"direction"`
	Kind      elevator.RequestKind `json:"kind"`
}
```

这个结构只表达“创建请求时客户端能提供的输入”。

### 创建请求的调用链

现在 `POST /api/request` 的主要流程是：

```text
1. 检查必须是 POST 方法
2. 解析 JSON 请求体
3. 校验 floor、direction、kind
4. 调用 System.AddRequest(floor, direction, kind)
5. 返回后端创建出来的 Request
```

返回示例大致是：

```json
{
  "status": "accepted",
  "currentTick": 0,
  "request": {
    "id": 1,
    "floor": 4,
    "direction": "up",
    "kind": "hall",
    "status": "pending",
    "createdTick": 0,
    "assignedTick": 0,
    "completedTick": 0,
    "assignedElevatorId": 0
  }
}
```

这里可以看到：客户端只传了 3 个字段，但后端返回的是完整 `Request`。

### 错误处理

当前错误响应先继续使用 Go 标准库的 `http.Error()`：

```go
http.Error(w, err.Error(), http.StatusBadRequest)
```

这种方式会返回普通文本错误。它不如统一 JSON 错误响应规范，但当前项目还处在模型和 API 快速调整阶段，先保持简单。

错误场景包括：

```text
不是 POST 方法
请求体不是合法 JSON
请求体为空
请求体包含未知字段，例如 id/status
floor 越界
kind 不是 hall/cabin
direction 不是 up/down/idle
hall request 使用 idle 方向
cabin request 使用非 idle 方向
```

其中“拒绝未知字段”由：

```go
decoder.DisallowUnknownFields()
```

实现。这样如果客户端提交：

```json
{
  "id": 99,
  "floor": 4,
  "direction": "up",
  "kind": "hall"
}
```

后端会返回 `400 Bad Request`，因为 `id` 不应该由客户端指定。

### 本次测试

新增了 `internal/api/handler_test.go`，覆盖：

```text
合法请求会创建后端拥有的 Request
客户端提交 id/status 会被拒绝
hall 请求不能使用 idle
cabin 请求不能使用 up/down
非法 JSON 会被拒绝
```

已运行：

```bash
GOCACHE=/tmp/os_sp26_proj1-go-build go test ./...
```

结果通过。
