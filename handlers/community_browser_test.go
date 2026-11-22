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

// Uses a single-connection, isolated PostgreSQL schema. Every write is a
// disposable fixture and no email/webhook provider is configured.
func TestCommunityBrowserJourney(t *testing.T) {
    if os.Getenv("L2E_BROWSER_E2E")!="1" {t.Skip("Browser journey runs in UI CI")}
    t.Setenv("JWT_SECRET","community-browser-fixture-secret")
    t.Setenv("DISCORD_WEBHOOK_URL","");t.Setenv("RESEND_API_KEY","")
    database:=timetableTestDB(t,false)
    var owner,learner,booking int64
    for i,target:=range []*int64{&owner,&learner} {
        if err:=database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES($1,$2,'fixture') RETURNING id`,[]string{"Community host","Community learner"}[i],fmt.Sprintf("community-%d@example.invalid",i)).Scan(target);err!=nil {t.Fatal(err)}
    }
    if err:=database.QueryRow(`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,status,meeting_type) VALUES($1,$2,CURRENT_DATE-1,'09:00','10:00','Completed API lesson','completed','online') RETURNING id`,owner,learner).Scan(&booking);err!=nil {t.Fatal(err)}
    router:=gin.New();router.Use(func(c *gin.Context){c.Set("db",database);c.Next()})
    router.LoadHTMLGlob("../templates/*.html");router.Static("/static","../static")
    for path,template:=range map[string]string{"projects":"projects","buddies":"buddy","masterclasses":"masterclass","my-bookings":"my-bookings"} {name:=template;router.GET("/"+path,func(c *gin.Context){c.HTML(200,name+".html",nil)})}
    group:=router.Group("/api/v1",middleware.AuthRequired)
    group.GET("/me",func(c *gin.Context){c.JSON(200,gin.H{"id":c.GetInt64("user_id")})})
    group.GET("/projects",ListProjects);group.POST("/projects",CreateProject);group.GET("/projects/:id",GetProject);group.PUT("/projects/:id/status",SetProjectStatus);group.POST("/projects/:id/requests",CreateProjectRequest)
    group.GET("/project-requests",ListProjectRequests);group.PUT("/project-requests/:id/status",RespondProjectRequest)
    group.GET("/collaboration/preferences",GetCollaborationPreference);group.PUT("/collaboration/preferences",SetCollaborationPreference);group.GET("/collaborators",ListCollaborators)
    group.GET("/buddies",ListBuddies);group.POST("/buddies/request",RequestBuddy);group.POST("/buddies/respond",RespondBuddy)
    group.GET("/masterclasses",ListMasterclasses);group.POST("/masterclasses",CreateMasterclass);group.POST("/masterclasses/:id/rsvp",ReserveMasterclass);group.DELETE("/masterclasses/:id/rsvp",CancelMasterclassReservation);group.PUT("/masterclasses/:id/cancel",CancelMasterclass)
    group.GET("/bookings",GetMyBookings);group.POST("/bookings/:id/reviews",CreateReview)
    server:=httptest.NewServer(router);defer server.Close()
    script,err:=filepath.Abs("../tests/ui/real-community.mjs");if err!=nil {t.Fatal(err)}
    command:=exec.Command("node",script);command.Env=append(os.Environ(),"L2E_TEST_ORIGIN="+server.URL,"L2E_OWNER_TOKEN="+reflectionTestToken(t,owner),"L2E_LEARNER_TOKEN="+reflectionTestToken(t,learner))
    if output,err:=command.CombinedOutput();err!=nil {t.Fatalf("community browser: %v\n%s",err,output)}else {t.Log(string(output))}
    var members,buddies,reviews int
    if err:=database.QueryRow(`SELECT count(*) FROM project_members WHERE user_id=$1`,learner).Scan(&members);err!=nil {t.Fatal(err)}
    if err:=database.QueryRow(`SELECT count(*) FROM buddy_connections WHERE status='active'`).Scan(&buddies);err!=nil {t.Fatal(err)}
    if err:=database.QueryRow(`SELECT count(*) FROM reviews WHERE booking_id=$1 AND would_book_again=TRUE`,booking).Scan(&reviews);err!=nil {t.Fatal(err)}
    if members!=1||buddies!=1||reviews!=1 {t.Fatalf("durable outcomes: members=%d buddies=%d reviews=%d",members,buddies,reviews)}
}
