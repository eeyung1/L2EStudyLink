package middleware

import (
    "context"
    "database/sql"
    "database/sql/driver"
    "errors"
    "io"
    "net/http"
    "net/http/httptest"
    "sync/atomic"
    "testing"

    "github.com/gin-gonic/gin"
)

type adminDriver struct{}
type adminConn struct{}
type adminRows struct{ values []driver.Value; done bool }
var adminResult atomic.Value

func (adminDriver) Open(string) (driver.Conn, error) { return adminConn{}, nil }
func (adminConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (adminConn) Close() error { return nil }
func (adminConn) Begin() (driver.Tx, error) { return nil, errors.New("unused") }
func (adminConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
    v := adminResult.Load().([]driver.Value)
    return &adminRows{values:v}, nil
}
func (r *adminRows) Columns() []string { return []string{"is_admin", "is_suspended"} }
func (r *adminRows) Close() error { return nil }
func (r *adminRows) Next(dest []driver.Value) error {
    if r.done || r.values == nil { return io.EOF }
    copy(dest, r.values)
    r.done = true
    return nil
}

func TestAdminRequired(t *testing.T) {
    sql.Register("admin-test", adminDriver{})
    db, err := sql.Open("admin-test", "")
    if err != nil { t.Fatal(err) }
    defer db.Close()
    gin.SetMode(gin.TestMode)
    cases := []struct { name string; values []driver.Value; want int }{
        {"admin", []driver.Value{true, false}, http.StatusOK},
        {"ordinary user", []driver.Value{false, false}, http.StatusForbidden},
        {"suspended admin", []driver.Value{true, true}, http.StatusForbidden},
        {"deleted admin", nil, http.StatusForbidden},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            adminResult.Store(tc.values)
            r := gin.New()
            r.Use(func(c *gin.Context) { c.Set("db", db); c.Set("user_id", int64(1)) })
            r.GET("/admin", AdminRequired, func(c *gin.Context) { c.Status(http.StatusOK) })
            w := httptest.NewRecorder()
            r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
            if w.Code != tc.want { t.Fatalf("status = %d, want %d", w.Code, tc.want) }
        })
    }
}
