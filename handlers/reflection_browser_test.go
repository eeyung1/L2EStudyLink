package handlers

import (
    "fmt"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "github.com/gin-gonic/gin"
    "L2EStudyLink/middleware"
)

// Opt-in browser journey against real Gin handlers and an isolated PostgreSQL schema.
func TestReflectionBrowserJourney(t *testing.T) {
    if os.Getenv("L2E_BROWSER_E2E")!="1" {t.Skip("Browser journey runs in the UI CI job")}
    t.Setenv("JWT_SECRET","browser-reflection-fixture-secret")
    t.Setenv("VOICE_UPLOAD_DIR",t.TempDir())
    database:=timetableTestDB(t,false);owner,block:=timetableFixture(t,database)
    var other int64
    if err:=database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Other browser learner','other-browser@example.invalid','fixture') RETURNING id`).Scan(&other);err!=nil {t.Fatal(err)}
    router:=privateReflectionRouter(database)
    router.LoadHTMLGlob("../templates/*.html");router.Static("/static","../static")
    router.GET("/timetable",func(c *gin.Context){c.HTML(200,"timetable.html",nil)})
    router.GET("/reflections",func(c *gin.Context){c.HTML(200,"reflections.html",nil)})
    router.GET("/api/v1/me",middleware.AuthRequired,func(c *gin.Context){c.JSON(200,gin.H{"id":c.GetInt64("user_id"),"name":"Fixture learner"})})
    router.GET("/api/v1/me/streaks",middleware.AuthRequired,func(c *gin.Context){c.JSON(200,gin.H{"current_streak":0,"longest_streak":0,"total_active_days":0,"daily_counts":gin.H{}})})
    server:=httptest.NewServer(router);defer server.Close()
    script,err:=filepath.Abs("../tests/ui/real-reflections.mjs");if err!=nil {t.Fatal(err)}
    command:=exec.Command("node",script)
    command.Env=append(os.Environ(),"L2E_TEST_ORIGIN="+server.URL,"L2E_TEST_OWNER_TOKEN="+reflectionTestToken(t,owner),"L2E_TEST_OTHER_TOKEN="+reflectionTestToken(t,other),fmt.Sprintf("L2E_TEST_BLOCK=%d",block))
    if result,err:=command.CombinedOutput();err!=nil {t.Fatalf("real browser reflection journey: %v\n%s",err,result)}else {t.Log(string(result))}
}
