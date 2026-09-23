package handlers

import (
    "database/sql"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
)

func TestEditSkillUpdatesOnlyOwner(t *testing.T) {
    dsn := os.Getenv("TEST_DATABASE_URL")
    if dsn == "" { t.Skip("TEST_DATABASE_URL not configured") }
    database, err := sql.Open("postgres", dsn)
    if err != nil { t.Fatal(err) }
    defer database.Close()
    schema, err := os.ReadFile("../schema.sql")
    if err != nil { t.Fatal(err) }
    if _,err=database.Exec(string(schema));err!=nil {t.Fatal(err)}
    var owner,other int64
    for i,id := range []*int64{&owner,&other} {
        err=database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES($1,$2,'test') RETURNING id`,"Skill tester",fmt.Sprintf("skill-edit-%d@example.com",i)).Scan(id)
        if err!=nil {t.Fatal(err)}
    }
    defer database.Exec(`DELETE FROM users WHERE id IN ($1,$2)`,owner,other)
    if _,err=database.Exec(`INSERT INTO skills(user_id,skill_name,proficiency) VALUES($1,'Go','beginner')`,owner);err!=nil {t.Fatal(err)}
    router:=gin.New()
    router.Use(func(c *gin.Context){c.Set("db",database);if c.GetHeader("X-Test-User")=="owner" {c.Set("user_id",owner)} else {c.Set("user_id",other)};c.Next()})
    router.PUT("/skills/:skill",EditSkill)
    request:=func(who,body string) int {w:=httptest.NewRecorder();req:=httptest.NewRequest(http.MethodPut,"/skills/Go",strings.NewReader(body));req.Header.Set("X-Test-User",who);req.Header.Set("Content-Type","application/json");router.ServeHTTP(w,req);return w.Code}
    if code:=request("other",`{"skill_name":"Rust","proficiency":"advanced"}`);code!=404 {t.Fatalf("other user changed skill: %d",code)}
    if code:=request("owner",`{"skill_name":"Go","proficiency":"expert"}`);code!=400 {t.Fatalf("invalid proficiency: %d",code)}
    if code:=request("owner",`{"skill_name":"Golang","proficiency":"advanced"}`);code!=200 {t.Fatalf("owner edit: %d",code)}
    var name,level string
    if err=database.QueryRow(`SELECT skill_name,proficiency FROM skills WHERE user_id=$1`,owner).Scan(&name,&level);err!=nil||name!="Golang"||level!="advanced" {t.Fatalf("updated skill: %s %s %v",name,level,err)}
}
