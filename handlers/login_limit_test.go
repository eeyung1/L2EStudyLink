package handlers

import (
 "database/sql"
 "fmt"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/gin-gonic/gin"
 "golang.org/x/crypto/bcrypt"
)

func testLoginThrottle(t *testing.T,db *sql.DB,user int64) {
 t.Setenv("JWT_SECRET","a-test-secret-not-used-in-production")
 hash,err:=bcrypt.GenerateFromPassword([]byte("right-password"),bcrypt.MinCost);if err!=nil {t.Fatal(err)}
 if _,err=db.Exec(`UPDATE users SET password_hash=$1 WHERE id=$2`,string(hash),user);err!=nil {t.Fatal(err)}
 defer db.Exec(`DELETE FROM login_attempts`)
 r:=gin.New();r.Use(func(c *gin.Context){c.Set("db",db);c.Next()});r.POST("/login",Login)
 call:=func(email,password string) int {t.Helper();w:=httptest.NewRecorder();body:=fmt.Sprintf(`{"email":%q,"password":%q}`,email,password);req:=httptest.NewRequest("POST","/login",strings.NewReader(body));req.Header.Set("Content-Type","application/json");r.ServeHTTP(w,req);return w.Code}
 for n:=0;n<5;n++ {if status:=call("project-test-0@example.com","wrong");status!=401 {t.Fatalf("failed login %d: %d",n,status)}}
 if status:=call("project-test-0@example.com","right-password");status!=429 {t.Fatalf("locked login: %d",status)}
 if status:=call("someone-else@example.com","wrong");status!=401 {t.Fatalf("separate account: %d",status)}
 if _,err=db.Exec(`UPDATE login_attempts SET window_started_at=NOW()-INTERVAL '16 minutes',locked_until=NOW()-INTERVAL '1 minute' WHERE attempts=5`);err!=nil {t.Fatal(err)}
 if status:=call("project-test-0@example.com","right-password");status!=200 {t.Fatalf("login after cooldown: %d",status)}
 w:=httptest.NewRecorder();req:=httptest.NewRequest("POST","/login",strings.NewReader(`{"email":"project-test-0@example.com","password":"right-password"}`));req.Header.Set("Content-Type","application/json");r.ServeHTTP(w,req)
 if w.Code!=200||strings.Contains(w.Body.String(),`"token"`)||!strings.Contains(w.Header().Get("Set-Cookie"),"HttpOnly") {t.Fatalf("cookie-only login: %d %s %s",w.Code,w.Body.String(),w.Header().Get("Set-Cookie"))}
 var count int;if err=db.QueryRow(`SELECT count(*) FROM login_attempts WHERE attempts=5`).Scan(&count);err!=nil||count!=0 {t.Fatalf("failed attempts not cleared: %d %v",count,err)}
}
