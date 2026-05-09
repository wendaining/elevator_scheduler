package elevator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// RequestStore 封装已完成请求的 SQLite 持久化逻辑。
// System 只需要调用它保存完成请求，不需要关心 SQL 语句细节。
type RequestStore struct {
	db *sql.DB
}

// OpenRequestStore 打开 SQLite 数据库，并确保 completed_requests 表存在。
func OpenRequestStore(databasePath string) (*RequestStore, error) {
	if databasePath == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}
	if err := ensureDatabaseDirectory(databasePath); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", databasePath)
	// 如果 databasePath 是 ":memory:"，SQLite 会在内存中创建一个临时数据库，
	// 程序结束后会自动销毁，不会留下文件。
	// 如果是 data/requests.db，则 SQLite 会把数据保存到这个文件
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &RequestStore{db: db}
	// initSchema 创建表结构，如果表已经存在则什么也不做
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

// Close 关闭底层数据库连接。
func (s *RequestStore) Close() error {
	// 这里的防御性编程我也不知道有什么用
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SaveCompletedRequest 将一个已经完成的 Request 写入 completed_requests 表。
func (s *RequestStore) SaveCompletedRequest(request Request) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("request store is not initialized")
	}

	_, err := s.db.Exec(`
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
	`, request.ID,
		request.Floor,
		request.Direction,
		request.Kind,
		request.Status,
		request.CreatedTick,
		request.AssignedTick,
		request.CompletedTick,
		request.AssignedElevatorID,
	)
	if err != nil {
		return fmt.Errorf("save completed request %d: %w", request.ID, err)
	}
	return nil
}

// CompletedRequestCount 返回数据库中已完成请求的数量，主要用于测试和调试。
func (s *RequestStore) CompletedRequestCount() (int, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("request store is not initialized")
	}

	var count int
	// Go 很喜欢 .Scan() 传指针作为输出的设计
	err := s.db.QueryRow(`SELECT COUNT(*) FROM completed_requests`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count completed requests: %w", err)
	}
	return count, nil
}

// MaxCompletedRequestID 返回数据库中已经保存过的最大请求 ID。
// 系统启动时用它恢复 nextRequestID，避免重启后从 1 重新编号。
func (s *RequestStore) MaxCompletedRequestID() (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("request store is not initialized")
	}

	var maxID sql.NullInt64
	//sql.NullInt64 可以表达两种状态：
	// Valid == true 时，Int64 字段包含有效值；
	// Valid == false 时，表示数据库中的值是 NULL，此时 Int64 字段的值应该被忽略。
	err := s.db.QueryRow(`SELECT MAX(id) FROM completed_requests`).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("read max completed request id: %w", err)
	}
	if !maxID.Valid {
		return 0, nil
	}
	return maxID.Int64, nil
}

// CompletedRequestByID 按请求 ID 读取一条已完成请求记录，主要用于测试。
func (s *RequestStore) CompletedRequestByID(requestID int64) (*Request, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("request store is not initialized")
	}

	var request Request
	err := s.db.QueryRow(`
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
	// requestID 是如何传入 `?` 的？
	// QueryRow 的 参数列表：(query String, args ...any)
	// 第二个可变参数依次匹配 query 中的 `?` 占位符，最终形成完整的 SQL 语句。
	if err != nil {
		return nil, fmt.Errorf("read completed request %d: %w", requestID, err)
	}
	return &request, nil
}

// initSchema 创建 completed_requests 表。字段和 Request 结构体保持一一对应。
func (s *RequestStore) initSchema() error {
	_, err := s.db.Exec(`
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
	`)
	if err != nil {
		return fmt.Errorf("init completed_requests schema: %w", err)
	}
	return nil
}

// ensureDatabaseDirectory 确保文件型 SQLite 数据库所在目录存在。
// 对 ":memory:" 这种内存数据库不需要创建目录。
func ensureDatabaseDirectory(databasePath string) error {
	if databasePath == ":memory:" {
		return nil
	}

	dir := filepath.Dir(databasePath)
	if dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create database directory %s: %w", dir, err)
	}
	return nil
}
