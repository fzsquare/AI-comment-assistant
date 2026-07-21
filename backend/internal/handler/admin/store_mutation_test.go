package admin

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ppk/backend/internal/config"
	"ppk/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type storeMutationSQLState struct {
	mu      sync.Mutex
	nextID  int64
	execSQL []string
	execArg [][]driver.NamedValue
}

var storeMutationDriverID atomic.Uint64

type storeMutationSQLDriver struct {
	state *storeMutationSQLState
}

func (d *storeMutationSQLDriver) Open(string) (driver.Conn, error) {
	return &storeMutationSQLConn{state: d.state}, nil
}

type storeMutationSQLConn struct {
	state *storeMutationSQLState
}

func (c *storeMutationSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (c *storeMutationSQLConn) Close() error { return nil }

func (c *storeMutationSQLConn) Begin() (driver.Tx, error) {
	return storeMutationSQLTx{}, nil
}

func (c *storeMutationSQLConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return storeMutationSQLTx{}, nil
}

func (c *storeMutationSQLConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	copied := append([]driver.NamedValue(nil), args...)
	c.state.execSQL = append(c.state.execSQL, query)
	c.state.execArg = append(c.state.execArg, copied)
	c.state.nextID++
	return storeMutationSQLResult{lastInsertID: c.state.nextID, rowsAffected: 1}, nil
}

func (c *storeMutationSQLConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "store_types") {
		now := time.Now()
		return &storeMutationSQLRows{
			columns: []string{"id", "code", "name", "industry_code", "is_preset", "status", "created_at", "updated_at"},
			values:  [][]driver.Value{{int64(7), "restaurant", "餐饮", "restaurant", true, int64(model.StatusEnabled), now, now}},
		}, nil
	}
	return &storeMutationSQLRows{columns: []string{"id"}}, nil
}

type storeMutationSQLTx struct{}

func (storeMutationSQLTx) Commit() error   { return nil }
func (storeMutationSQLTx) Rollback() error { return nil }

type storeMutationSQLResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r storeMutationSQLResult) LastInsertId() (int64, error) { return r.lastInsertID, nil }
func (r storeMutationSQLResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type storeMutationSQLRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *storeMutationSQLRows) Columns() []string { return r.columns }
func (r *storeMutationSQLRows) Close() error      { return nil }
func (r *storeMutationSQLRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func TestNormalizeStoreMutationRequiresPasswordWhenCreating(t *testing.T) {
	req := storeMutationRequest{
		Account:   "merchant-new",
		TypeID:    1,
		StoreName: "新店",
	}

	if err := normalizeStoreMutationRequest(&req, true); err == nil {
		t.Fatal("expected missing password to fail when creating store")
	}
}

func TestNormalizeStoreMutationAllowsBlankPasswordWhenEditing(t *testing.T) {
	req := storeMutationRequest{
		Account:              "  merchant-new  ",
		TypeID:               1,
		StoreName:            "  新店  ",
		PrimaryPlatformStyle: "",
	}

	if err := normalizeStoreMutationRequest(&req, false); err != nil {
		t.Fatalf("expected blank password to be allowed when editing, got %v", err)
	}
	if req.Account != "merchant-new" {
		t.Fatalf("account got %q, want trimmed merchant-new", req.Account)
	}
	if req.StoreName != "新店" {
		t.Fatalf("store name got %q, want trimmed 新店", req.StoreName)
	}
	if req.PrimaryPlatformStyle != "dianping" {
		t.Fatalf("platform got %q, want default dianping", req.PrimaryPlatformStyle)
	}
}

func TestNormalizeStoreMutationAllowsBlankPlatformURL(t *testing.T) {
	req := storeMutationRequest{
		Account:   "merchant-new",
		Password:  "123456",
		TypeID:    1,
		StoreName: "新店",
	}

	if err := normalizeStoreMutationRequest(&req, true); err != nil {
		t.Fatalf("expected blank platform url to be allowed, got %v", err)
	}
}

func TestNormalizeStoreMutationRejectsUnsupportedPlatformURL(t *testing.T) {
	req := storeMutationRequest{
		Account:     "merchant-new",
		Password:    "123456",
		TypeID:      1,
		StoreName:   "新店",
		PlatformURL: "javascript:alert(1)",
	}

	if err := normalizeStoreMutationRequest(&req, true); err == nil {
		t.Fatal("expected unsupported platform url to fail")
	}
}

func TestInitialReviewGenerationTaskUsesPrimaryPlatformAndDefaultCount(t *testing.T) {
	store := model.Store{ID: 42, PrimaryPlatformStyle: "meituan"}

	task := initialReviewGenerationTask(store, 10)

	if task.StoreID != store.ID {
		t.Fatalf("store id got %d, want %d", task.StoreID, store.ID)
	}
	if task.PlatformStyle != "meituan" {
		t.Fatalf("platform got %q, want meituan", task.PlatformStyle)
	}
	if task.TriggerType != model.TriggerInit {
		t.Fatalf("trigger got %q, want %q", task.TriggerType, model.TriggerInit)
	}
	if task.TargetCount != 10 {
		t.Fatalf("target count got %d, want 10", task.TargetCount)
	}
	if task.Status != model.TaskStatusPending {
		t.Fatalf("status got %q, want %q", task.Status, model.TaskStatusPending)
	}
	if task.DuplicateCheckVersion == "" {
		t.Fatal("duplicate check version must be recorded when the task is queued")
	}
}

func TestCreateStorePersistsInitialReviewGenerationTask(t *testing.T) {
	state := &storeMutationSQLState{nextID: 100}
	driverName := fmt.Sprintf("store-mutation-review-task-%d", storeMutationDriverID.Add(1))
	sql.Register(driverName, &storeMutationSQLDriver{state: state})
	sqlDB, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open recording database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("open gorm database: %v", err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/stores", strings.NewReader(`{
		"account":"merchant-init-task",
		"password":"secret123",
		"typeId":7,
		"storeName":"自动生成测试店",
		"primaryPlatformStyle":"meituan"
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h := Handler{
		DB: db,
		Config: config.Config{
			DefaultReviewTargetCount: 10,
		},
	}
	h.createStore(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("create store status got %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	var taskValues string
	for i, query := range state.execSQL {
		if strings.Contains(query, "review_generation_tasks") {
			parts := make([]string, 0, len(state.execArg[i]))
			for _, arg := range state.execArg[i] {
				parts = append(parts, driverValueString(arg.Value))
			}
			taskValues = strings.Join(parts, "|")
			break
		}
	}
	if taskValues == "" {
		t.Fatalf("create store did not insert review_generation_tasks; SQL=%#v", state.execSQL)
	}
	for _, expected := range []string{"meituan", model.TriggerInit, "10", model.TaskStatusPending} {
		if !strings.Contains(taskValues, expected) {
			t.Fatalf("initial task values %q missing %q", taskValues, expected)
		}
	}
}

func driverValueString(value interface{}) string {
	return fmt.Sprint(value)
}

func TestDeriveNFCCardStatusRequiresRealBoundTag(t *testing.T) {
	store := model.Store{ID: 1, UUID: "store-uuid", Status: model.StatusEnabled}

	status := deriveNFCCardStatus(store, 0, 0, 0)
	if status.PrimaryStatus != "unwritten" || status.RouteStatus != "no_bound_tag" {
		t.Fatalf("empty tag status got %#v, want unwritten/no_bound_tag", status)
	}

	status = deriveNFCCardStatus(store, 1, 1, 0)
	if status.PrimaryStatus != "usable" || status.RouteStatus != "ok" || status.WrittenCount != 1 {
		t.Fatalf("bound tag status got %#v, want usable/ok with written count", status)
	}
}

func TestDeriveNFCCardStatusMarksInactiveStoreUnusable(t *testing.T) {
	store := model.Store{ID: 1, UUID: "store-uuid", Status: model.StatusDisabled}

	status := deriveNFCCardStatus(store, 1, 1, 0)
	if status.PrimaryStatus != "unusable" || status.RouteStatus != "store_inactive" {
		t.Fatalf("inactive store status got %#v, want unusable/store_inactive", status)
	}
}
