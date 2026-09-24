package handlers

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http/httptest"
    "os"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"

    projectdb "L2EStudyLink/db"
)

func TestProjectCollaborationFlow(t *testing.T) {
    dsn := os.Getenv("TEST_DATABASE_URL")
    if dsn == "" { t.Skip("TEST_DATABASE_URL not configured") }
    db, err := sql.Open("postgres", dsn)
    if err != nil { t.Fatal(err) }
    defer db.Close()
    schema, err := os.ReadFile("../schema.sql")
    if err != nil { t.Fatal(err) }
    // Render starts against an existing database. Exercise its repeatable
    // startup migration with only the original users table present.
    usersSchema := string(schema)[:strings.Index(string(schema), "-- Skills table")]
    if _, err := db.Exec(usersSchema); err != nil { t.Fatal(err) }
    t.Setenv("DATABASE_URL", dsn)
    if err := projectdb.InitDB(); err != nil { t.Fatalf("startup migration: %v", err) }
    defer projectdb.CloseDB()
    for _, table := range []string{"projects", "project_requests", "project_members", "collaboration_preferences"} {
        var exists bool
        if err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil || !exists { t.Fatalf("startup table %s: %v", table, err) }
    }
    if _, err := db.Exec(string(schema)); err != nil { t.Fatal(err) }
    var owner, applicant, invited int64
    for i, target := range []*int64{&owner, &applicant, &invited} {
        err = db.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES($1,$2,'test') RETURNING id`, fmt.Sprintf("Fellow %d",i),fmt.Sprintf("project-test-%d@example.com",i)).Scan(target)
        if err != nil { t.Fatal(err) }
    }
    defer db.Exec(`DELETE FROM users WHERE id IN ($1,$2,$3)`,owner,applicant,invited)
    t.Run("product email consent limits export",func(t *testing.T){
        if _,err:=db.Exec(`UPDATE users SET marketing_opt_in_at=NOW() WHERE id IN ($1,$2)`,owner,applicant);err!=nil {t.Fatal(err)}
        if _,err:=db.Exec(`UPDATE users SET is_suspended=TRUE WHERE id=$1`,applicant);err!=nil {t.Fatal(err)}
        defer db.Exec(`UPDATE users SET marketing_opt_in_at=NULL,is_suspended=FALSE WHERE id IN ($1,$2,$3)`,owner,applicant,invited)
        r:=gin.New()
        r.Use(func(c *gin.Context){c.Set("db",db);c.Set("user_id",owner);c.Next()})
        r.GET("/audience",ProductEmailAudience)
        r.GET("/audience.csv",ExportProductEmailAudience)
        r.PUT("/preference",SetProductEmails)
        request:=func(method,path,body string)*httptest.ResponseRecorder{
            w:=httptest.NewRecorder();req:=httptest.NewRequest(method,path,strings.NewReader(body));req.Header.Set("Content-Type","application/json");r.ServeHTTP(w,req);return w
        }
        if w:=request("GET","/audience","");w.Code!=200||!strings.Contains(w.Body.String(),`"eligible":1`) {t.Fatalf("audience: %d %s",w.Code,w.Body.String())}
        if w:=request("GET","/audience.csv","");w.Code!=200||w.Body.String()!="EMAIL\nproject-test-0@example.com\n" {t.Fatalf("export: %d %q",w.Code,w.Body.String())}
        if w:=request("PUT","/preference",`{"enabled":false}`);w.Code!=200 {t.Fatalf("disable: %d %s",w.Code,w.Body.String())}
        if w:=request("GET","/audience","");w.Code!=200||!strings.Contains(w.Body.String(),`"eligible":0`) {t.Fatalf("disabled audience: %d %s",w.Code,w.Body.String())}
    })
    t.Run("availability sequence follows existing IDs",func(t *testing.T){
        var highest int64
        if err:=db.QueryRow(`SELECT COALESCE(MAX(id),0)+1000 FROM availability`).Scan(&highest);err!=nil {t.Fatal(err)}
        if _,err:=db.Exec(`INSERT INTO availability(id,user_id,day_of_week,start_time,end_time) VALUES($1,$2,1,'07:15','08:45')`,highest,owner);err!=nil {t.Fatal(err)}
        defer db.Exec(`DELETE FROM availability WHERE id=$1`,highest)
        if _,err:=db.Exec(`SELECT setval(pg_get_serial_sequence('availability','id'),1,false)`);err!=nil {t.Fatal(err)}
        if err:=projectdb.RepairAvailabilitySequence(context.Background());err!=nil {t.Fatal(err)}
        var created int64
        if err:=db.QueryRow(`INSERT INTO availability(user_id,day_of_week,start_time,end_time) VALUES($1,2,'09:15','10:45') RETURNING id`,owner).Scan(&created);err!=nil {t.Fatalf("insert after sequence repair: %v",err)}
        defer db.Exec(`DELETE FROM availability WHERE id=$1`,created)
        if created<=highest {t.Fatalf("new ID %d did not follow existing %d",created,highest)}
    })
    t.Run("availability reads stored hours", func(t *testing.T) {
        if _,err:=db.Exec(`INSERT INTO availability(user_id,day_of_week,start_time,end_time) VALUES($1,0,'09:00','11:00')`,owner);err!=nil {t.Fatal(err)}
        defer db.Exec(`DELETE FROM availability WHERE user_id=$1`,owner)
        r:=gin.New()
        r.Use(func(c *gin.Context){c.Set("db",db);c.Set("user_id",owner);c.Next()})
        r.GET("/availability",GetAvailability)
        response:=httptest.NewRecorder()
        r.ServeHTTP(response,httptest.NewRequest("GET","/availability",nil))
        if response.Code!=200 {t.Fatalf("availability response: %d %s",response.Code,response.Body.String())}
        var slots []AvailabilitySlot
        if err:=json.Unmarshal(response.Body.Bytes(),&slots);err!=nil {t.Fatal(err)}
        if len(slots)!=1||slots[0].DayOfWeek!=0||slots[0].StartTime!="09:00"||slots[0].EndTime!="11:00" {t.Fatalf("unexpected availability: %+v",slots)}
    })
    t.Run("availability saves minutes atomically",func(t *testing.T){
        r:=gin.New();r.Use(func(c *gin.Context){c.Set("db",db);c.Set("user_id",owner);c.Next()});r.PUT("/availability",SetAvailability);r.GET("/availability",GetAvailability)
        put:=func(body string) int {w:=httptest.NewRecorder();req:=httptest.NewRequest("PUT","/availability",strings.NewReader(body));req.Header.Set("Content-Type","application/json");r.ServeHTTP(w,req);return w.Code}
        if code:=put(`[{"day_of_week":2,"start_time":"09:15","end_time":"10:45"}]`);code!=200 {t.Fatalf("minute slot: %d",code)}
        if code:=put(`[{"day_of_week":2,"start_time":"10:45","end_time":"09:15"}]`);code!=400 {t.Fatalf("invalid slot: %d",code)}
        w:=httptest.NewRecorder();r.ServeHTTP(w,httptest.NewRequest("GET","/availability",nil))
        var slots []AvailabilitySlot;if err:=json.Unmarshal(w.Body.Bytes(),&slots);err!=nil {t.Fatal(err)}
        if len(slots)!=1||slots[0].StartTime!="09:15"||slots[0].EndTime!="10:45" {t.Fatalf("saved minute slot lost: %+v",slots)}
        db.Exec(`DELETE FROM availability WHERE user_id=$1`,owner)
    })
    t.Run("skill edit belongs to owner", func(t *testing.T) {
        if _,err:=db.Exec(`INSERT INTO skills(user_id,skill_name,proficiency) VALUES($1,'Go','beginner')`,owner);err!=nil {t.Fatal(err)}
        r:=gin.New()
        r.Use(func(c *gin.Context){c.Set("db",db);if c.GetHeader("X-Test-User")=="owner" {c.Set("user_id",owner)} else {c.Set("user_id",applicant)};c.Next()})
        r.PUT("/skills/:skill",EditSkill)
        request:=func(who,body string) int {w:=httptest.NewRecorder();req:=httptest.NewRequest("PUT","/skills/Go",strings.NewReader(body));req.Header.Set("X-Test-User",who);req.Header.Set("Content-Type","application/json");r.ServeHTTP(w,req);return w.Code}
        if code:=request("other",`{"skill_name":"Rust","proficiency":"advanced"}`);code!=404 {t.Fatalf("other user changed skill: %d",code)}
        if code:=request("owner",`{"skill_name":"Go","proficiency":"expert"}`);code!=400 {t.Fatalf("invalid proficiency: %d",code)}
        if code:=request("owner",`{"skill_name":"Golang","proficiency":"advanced"}`);code!=200 {t.Fatalf("owner edit: %d",code)}
        var name,level string
        if err:=db.QueryRow(`SELECT skill_name,proficiency FROM skills WHERE user_id=$1`,owner).Scan(&name,&level);err!=nil||name!="Golang"||level!="advanced" {t.Fatalf("updated skill: %s %s %v",name,level,err)}
    })


    router := gin.New()
    router.Use(func(c *gin.Context) { var uid int64; fmt.Sscan(c.GetHeader("X-Test-User"),&uid); c.Set("user_id",uid); c.Set("db",projectdb.DB); c.Next() })
    router.POST("/projects",CreateProject)
    router.POST("/signup",Signup)
    router.GET("/projects",ListProjects)
    router.POST("/projects/:id/requests",CreateProjectRequest)
    router.PUT("/project-requests/:id/status",RespondProjectRequest)
    router.PUT("/collaboration/preferences",SetCollaborationPreference)
    router.GET("/collaborators",ListCollaborators)
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
    code, duplicateAccount := call(owner,"POST","/signup",`{"name":"Duplicate","email":"project-test-0@example.com","password":"password123"}`)
    if code != 409 { t.Fatalf("duplicate signup: %d %v",code,duplicateAccount) }
    projectsResponse:=httptest.NewRecorder()
    projectsRequest:=httptest.NewRequest("GET","/projects",nil)
    projectsRequest.Header.Set("X-Test-User",fmt.Sprint(applicant))
    router.ServeHTTP(projectsResponse,projectsRequest)
    if projectsResponse.Code!=200 {t.Fatalf("list open projects: %d %s",projectsResponse.Code,projectsResponse.Body.String())}
    var visibleProjects []struct { ID int64 `json:"id"`; Title string `json:"title"` }
    if err:=json.Unmarshal(projectsResponse.Body.Bytes(),&visibleProjects);err!=nil {t.Fatal(err)}
    if len(visibleProjects)!=1 || visibleProjects[0].ID!=int64(project["id"].(float64)) {t.Fatalf("open project not visible to another fellow: %+v",visibleProjects)}
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
    if _,err:=db.Exec(`INSERT INTO skills(user_id,skill_name,proficiency) VALUES($1,'Design','advanced')`,invited);err!=nil {t.Fatal(err)}
    if _,err:=db.Exec(`INSERT INTO availability(user_id,day_of_week,start_time,end_time) VALUES($1,1,'10:00','12:00')`,invited);err!=nil {t.Fatal(err)}
    fellowsResponse:=httptest.NewRecorder()
    fellowsRequest:=httptest.NewRequest("GET","/collaborators",nil)
    fellowsRequest.Header.Set("X-Test-User",fmt.Sprint(owner))
    router.ServeHTTP(fellowsResponse,fellowsRequest)
    if fellowsResponse.Code!=200 {t.Fatalf("list opted-in fellows: %d %s",fellowsResponse.Code,fellowsResponse.Body.String())}
    var fellows []struct { ID int64 `json:"id"`; Skills string `json:"skills"`; Availability []struct { Day int `json:"day_of_week"`; Start string `json:"start_time"` } `json:"availability"` }
    if err:=json.Unmarshal(fellowsResponse.Body.Bytes(),&fellows);err!=nil {t.Fatal(err)}
    if len(fellows)!=1 || fellows[0].ID!=invited || fellows[0].Skills!="Design" || len(fellows[0].Availability)!=1 || fellows[0].Availability[0].Start!="10:00" {t.Fatalf("unexpected opted-in fellows: %+v",fellows)}
    code, _ = call(owner,"POST",path,invite)
    if code != 201 { t.Fatalf("invite after opt-in: %d",code) }
    if err := db.QueryRow(`SELECT id FROM project_requests WHERE user_id=$1`,invited).Scan(&requestID);err!=nil{t.Fatal(err)}
    code, _ = call(invited,"PUT",fmt.Sprintf("/project-requests/%d/status",requestID),`{"status":"accepted"}`)
    if code != 200 { t.Fatalf("accept invitation: %d",code) }
    var count int
    if err := db.QueryRow(`SELECT count(*) FROM project_members WHERE user_id=$1`,invited).Scan(&count);err!=nil || count!=1 { t.Fatalf("invited membership: %v, %d",err,count) }
    t.Run("booking outcomes and reviews",func(t *testing.T){testBookingOutcomeAndReview(t,projectdb.DB,owner,applicant,invited)})
    t.Run("login throttling",func(t *testing.T){testLoginThrottle(t,projectdb.DB,owner)})
    t.Run("new signup clears prior guesses",func(t *testing.T){testSignupClearsPreRegistrationAttempts(t,projectdb.DB)})
}
