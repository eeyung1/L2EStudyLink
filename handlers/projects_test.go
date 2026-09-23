package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http/httptest"
    "os"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
)

func TestProjectCollaborationFlow(t *testing.T) {
    dsn := os.Getenv("TEST_DATABASE_URL")
    if dsn == "" { t.Skip("TEST_DATABASE_URL not configured") }
    db, err := sql.Open("postgres", dsn)
    if err != nil { t.Fatal(err) }
    defer db.Close()
    schema, err := os.ReadFile("../schema.sql")
    if err != nil { t.Fatal(err) }
    if _, err := db.Exec(string(schema)); err != nil { t.Fatal(err) }
    var owner, applicant, invited int64
    for i, target := range []*int64{&owner, &applicant, &invited} {
        err = db.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES($1,$2,'test') RETURNING id`, fmt.Sprintf("Fellow %d",i),fmt.Sprintf("project-test-%d@example.com",i)).Scan(target)
        if err != nil { t.Fatal(err) }
    }
    defer db.Exec(`DELETE FROM users WHERE id IN ($1,$2,$3)`,owner,applicant,invited)

    router := gin.New()
    router.Use(func(c *gin.Context) { var uid int64; fmt.Sscan(c.GetHeader("X-Test-User"),&uid); c.Set("user_id",uid); c.Set("db",db); c.Next() })
    router.POST("/projects",CreateProject)
    router.POST("/projects/:id/requests",CreateProjectRequest)
    router.PUT("/project-requests/:id/status",RespondProjectRequest)
    router.PUT("/collaboration/preferences",SetCollaborationPreference)
    router.GET("/projects/:id",GetProject)
    call := func(user int64, method,path,body string) (int,map[string]interface{}) {
        t.Helper()
        request := httptest.NewRequest(method,path,strings.NewReader(body))
        request.Header.Set("Content-Type","application/json")
        request.Header.Set("X-Test-User",fmt.Sprint(user))
        response := httptest.NewRecorder(); router.ServeHTTP(response,request)
        var payload map[string]interface{}
        if err := json.Unmarshal(response.Body.Bytes(),&payload); err != nil { t.Fatalf("%s %s: %v: %s",method,path,err,response.Body.String()) }
        return response.Code,payload
    }
    code, project := call(owner,"POST","/projects",`{"title":"Peer planner","description":"A planner built with fellow students","roles_needed":"Go developer","time_commitment":"3 hours weekly"}`)
    if code != 201 { t.Fatalf("create project: %d %v",code,project) }
    path := fmt.Sprintf("/projects/%.0f/requests",project["id"].(float64))
    code, _ = call(applicant,"POST",path,`{"kind":"application","note":"I can build the Go API"}`)
    if code != 201 { t.Fatalf("apply: %d",code) }
    code, duplicate := call(applicant,"POST",path,`{"kind":"application","note":"Another request from me"}`)
    if code != 409 { t.Fatalf("duplicate application: %d %v",code,duplicate) }
    var requestID int64
    if err := db.QueryRow(`SELECT id FROM project_requests WHERE user_id=$1`,applicant).Scan(&requestID);err!=nil{t.Fatal(err)}
    responsePath := fmt.Sprintf("/project-requests/%d/status",requestID)
    code, _ = call(invited,"PUT",responsePath,`{"status":"accepted"}`)
    if code != 403 { t.Fatalf("unauthorized acceptance: %d",code) }
    code, _ = call(owner,"PUT",responsePath,`{"status":"accepted"}`)
    if code != 200 { t.Fatalf("owner acceptance: %d",code) }
    code, details := call(applicant,"GET",fmt.Sprintf("/projects/%.0f",project["id"].(float64)),"")
    if code != 200 || len(details["members"].([]interface{})) != 1 { t.Fatalf("membership: %d %v",code,details) }
    invite := fmt.Sprintf(`{"kind":"invitation","user_id":%d,"note":"Build the UI"}`,invited)
    code, _ = call(owner,"POST",path,invite)
    if code != 403 { t.Fatalf("invite without opt-in: %d",code) }
    code, _ = call(invited,"PUT","/collaboration/preferences",`{"open_to_invites":true}`)
    if code != 200 { t.Fatalf("opt in: %d",code) }
    code, _ = call(owner,"POST",path,invite)
    if code != 201 { t.Fatalf("invite after opt-in: %d",code) }
    if err := db.QueryRow(`SELECT id FROM project_requests WHERE user_id=$1`,invited).Scan(&requestID);err!=nil{t.Fatal(err)}
    code, _ = call(invited,"PUT",fmt.Sprintf("/project-requests/%d/status",requestID),`{"status":"accepted"}`)
    if code != 200 { t.Fatalf("accept invitation: %d",code) }
    var count int
    if err := db.QueryRow(`SELECT count(*) FROM project_members WHERE user_id=$1`,invited).Scan(&count);err!=nil || count!=1 { t.Fatalf("invited membership: %v, %d",err,count) }
}
