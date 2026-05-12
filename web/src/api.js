/**
 * API 通信层，封装所有对后端 REST API 的调用。
 *
 * fetchState() 用于轮询全量系统状态，其他函数用于提交操作。
 * 所有函数都返回 Promise，调用方用 async/await 或 .then() 处理。
 */

const BASE = '/api'

/**
 * 获取前端运行所需的后端配置。
 * autoStepIntervalMs 是后端自动 Step 的间隔，前端轮询应使用同一个值。
 */
export async function fetchConfig() {
  const response = await fetch(`${BASE}/config`)
  if (!response.ok) {
    throw new Error(`GET /api/config failed: ${response.status}`)
  }
  return response.json()
}

/**
 * 获取电梯系统当前全量状态。
 * 返回的 JSON 结构见 model.go 的 json tag。
 */
export async function fetchState() {
  const response = await fetch(`${BASE}/state`)
  if (!response.ok) {
    throw new Error(`GET /api/state failed: ${response.status}`)
  }
  return response.json()
}

/**
 * 创建一个乘梯请求。
 * @param {number} floor      — 目标楼层
 * @param {string} direction   — "up" | "down" | "idle"
 * @param {string} kind        — "hall" | "cabin"
 * @param {number} elevatorId  — cabin 请求的电梯 ID（hall 请求传 0 或不传）
 */
export async function createRequest(floor, direction, kind, elevatorId = 0) {
  const response = await fetch(`${BASE}/request`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ floor, direction, kind, elevatorId }),
  })
  if (!response.ok) {
    const text = await response.text()
    throw new Error(`POST /api/request failed (${response.status}): ${text}`)
  }
  return response.json()
}

/**
 * 切换调度算法。
 * @param {string} name — "first-available" | "nearest-idle" | "fcfs" | "scan"
 */
export async function setScheduler(name) {
  const response = await fetch(`${BASE}/scheduler`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!response.ok) {
    throw new Error(`POST /api/scheduler failed: ${response.status}`)
  }
  return response.json()
}

/**
 * 设置楼层总数。
 * @param {number} floorCount — 范围 [2, 40]
 */
export async function setFloorCount(floorCount) {
  const response = await fetch(`${BASE}/floor-count`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ floorCount }),
  })
  if (!response.ok) {
    throw new Error(`POST /api/floor-count failed: ${response.status}`)
  }
  return response.json()
}

/**
 * 设置电梯总数。
 * @param {number} elevatorCount — 范围 [1, 10]
 */
export async function setElevatorCount(elevatorCount) {
  const response = await fetch(`${BASE}/elevator-count`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ elevatorCount }),
  })
  if (!response.ok) {
    throw new Error(`POST /api/elevator-count failed: ${response.status}`)
  }
  return response.json()
}
