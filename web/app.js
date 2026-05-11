// 这个文件是一个很小的“通信调试页”。
// 它刻意使用原生 DOM API 和 fetch，这样在迁移到 Vue 前，
// 你可以直接看清楚前端和后端的数据流。

// 这里暂时和后端 NewSystem(20, 5) 保持一致。
// 后续前端应该从 GET /api/state 读取 floorCount，而不是写死。
const floorCount = 20;

// document.querySelector 会用 CSS 选择器找到一个已有 DOM 节点。
// 这些节点在 index.html 里只是空容器，本文件负责把 UI 填进去。
const floorList = document.querySelector("#floorList");
const elevatorList = document.querySelector("#elevatorList");
const pendingRequests = document.querySelector("#pendingRequests");
const statusText = document.querySelector("#statusText");
const schedulerText = document.querySelector("#schedulerText");
const hallModeButton = document.querySelector("#hallModeButton");
const cabinModeButton = document.querySelector("#cabinModeButton");

// 当前前端处于哪种请求模式：
// - hall：楼层外部按钮，请求包含 up/down 方向。
// - cabin：电梯内部按钮，请求表示“选择一个目标楼层”。
let currentRequestKind = "hall";

function createFloorButtons() {
  floorList.replaceChildren();

  // 从高楼层往低楼层创建按钮，这样页面上最高楼层显示在最上面。
  for (let floor = floorCount; floor >= 1; floor -= 1) {
    // document.createElement 会在内存里创建一个新的 DOM 节点。
    // 只有 append 到页面已有节点之后，它才会真正显示出来。
    const row = document.createElement("div");
    row.className = "floor-row";

    const label = document.createElement("span");
    label.className = "floor-label";
    label.textContent = `${floor}F`;

    if (currentRequestKind === "cabin") {
      const selectButton = document.createElement("button");
      selectButton.className = "wide-button";
      selectButton.textContent = "Select";
      // Cabin 请求来自电梯内部，用户只选择目标楼层，不选择上下方向。
      // 当前后端模型仍要求 Direction 字段，所以先传 idle。
      selectButton.addEventListener("click", () =>
        submitRequest(floor, "idle", "cabin"),
      );

      row.append(label, selectButton);
      floorList.append(row);
      continue;
    }

    const upButton = document.createElement("button");
    upButton.textContent = "Up";
    // 最高层不能继续上行，所以禁用 Up 按钮。
    upButton.disabled = floor === floorCount;
    // addEventListener 用来注册用户点击后要执行的代码。
    // 这里的箭头函数会记住当前按钮对应的 floor。
    upButton.addEventListener("click", () =>
      submitRequest(floor, "up", "hall"),
    );

    const downButton = document.createElement("button");
    downButton.textContent = "Down";
    // 1 楼不能继续下行，所以禁用 Down 按钮。
    downButton.disabled = floor === 1;
    downButton.addEventListener("click", () =>
      submitRequest(floor, "down", "hall"),
    );

    // append 可以一次性追加多个子节点。
    row.append(label, upButton, downButton);
    floorList.append(row);
  }
}

function setRequestKind(kind) {
  currentRequestKind = kind;

  hallModeButton.classList.toggle("active", kind === "hall");
  cabinModeButton.classList.toggle("active", kind === "cabin");

  createFloorButtons();
}

async function fetchState() {
  // fetch 会从浏览器向 Go 后端发送 HTTP 请求。
  // 如果不指定 method，fetch 默认使用 GET。
  const response = await fetch("/api/state");
  if (!response.ok) {
    throw new Error(`GET /api/state failed: ${response.status}`);
  }
  // response.json() 会把响应体按 JSON 解析成 JavaScript 对象。
  return response.json();
}

async function submitRequest(floor, direction, kind) {
  try {
    // POST /api/request 会在后端创建一个新的乘梯请求。
    const response = await fetch("/api/request", {
      method: "POST",
      headers: {
        // 这个请求头告诉 Go 的 json.Decoder：请求体是 JSON。
        "Content-Type": "application/json",
      },
      // HTTP 请求体本质上是文本字节。JSON.stringify 会把 JavaScript 对象
      // 转成 Go 的 handleRequest 期望收到的 JSON 字符串。
      body: JSON.stringify({
        floor: floor,
        direction: direction,
        kind: kind,
      }),
    });

    if (!response.ok) {
      const message = await response.text();
      throw new Error(message);
    }

    // POST 成功后重新读取最新状态，这样页面能立刻更新。
    await refreshState();
  } catch (error) {
    statusText.textContent = error.message;
  }
}

function renderState(state) {
  schedulerText.textContent = `Scheduler: ${state.schedulerName || "-"}`;

  // 渲染最新状态前，先移除所有旧的电梯卡片。
  // 对这个临时原生 JS 页面来说，这种做法简单且足够。
  elevatorList.replaceChildren();

  // state.elevators 来自后端 System.Snapshot() 返回的 JSON。
  for (const elevator of state.elevators) {
    const card = document.createElement("article");
    card.className = "elevator-card";

    card.append(
      createField(`E${elevator.id}`, "Elevator"),
      createField(elevator.currentFloor, "Floor"),
      createField(elevator.direction, "Direction"),
      createField(elevator.doorOpen ? "Open" : "Closed", "Door"),
      createField(formatStops(elevator.stops), "Stops"),
    );

    elevatorList.append(card);
  }

  // 把待处理请求格式化成 JSON，方便学习和调试时查看。
  pendingRequests.textContent = JSON.stringify(state.requests || {}, null, 2);
}

function formatStops(stops) {
  if (!stops || stops.length === 0) {
    return "-";
  }

  return stops
    .map((stop) => `${stop.floor}F/${stop.reason}`)
    .join(", ");
}

function createField(value, label) {
  const field = document.createElement("div");
  field.className = label === "Elevator" ? "elevator-name" : "elevator-field";

  if (label === "Elevator") {
    field.textContent = value;
    return field;
  }

  // 小标签使用文本节点，真正的值使用 <strong> 节点，
  // 这样 CSS 可以把值显示得更醒目。
  const labelNode = document.createTextNode(label);
  const valueNode = document.createElement("strong");
  valueNode.textContent = value;

  field.append(labelNode, valueNode);
  return field;
}

async function refreshState() {
  try {
    // GET /api/state 是前后端循环里的“读取状态”步骤。
    const state = await fetchState();
    renderState(state);
    statusText.textContent = "Connected";
  } catch (error) {
    statusText.textContent = error.message;
  }
}

async function tick() {
  try {
    // 系统时间由 Go 后端的后台 ticker 自动推进。
    // 前端定时读取状态即可，不再手动调用 Step API。
    await refreshState();
  } catch (error) {
    statusText.textContent = error.message;
  }
}

// 页面启动流程。
hallModeButton.addEventListener("click", () => setRequestKind("hall"));
cabinModeButton.addEventListener("click", () => setRequestKind("cabin"));
createFloorButtons();
refreshState();
// setInterval 每 800ms 调用一次 tick，因此可以看到电梯逐步移动。
setInterval(tick, 800);
