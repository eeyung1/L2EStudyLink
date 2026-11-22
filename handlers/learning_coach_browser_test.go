package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/stdlib"
    "L2EStudyLink/middleware"
)

func TestLearningCoachBrowserJourney(t *testing.T) {
    if os.Getenv("L2E_BROWSER_E2E")!="1" {t.Skip("Browser journey runs in UI CI")}
    t.Setenv("JWT_SECRET","learning-coach-browser-fixture-secret")
    t.Setenv("GROQ_API_KEY","fixture-only-no-network")
    t.Setenv("DISCORD_WEBHOOK_URL","");t.Setenv("RESEND_API_KEY","")
    fixture:=timetableTestDB(t,false)
    var schema string
    if err:=fixture.QueryRow(`SELECT current_schema()`).Scan(&schema);err!=nil {t.Fatal(err)}
    config,err:=pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"));if err!=nil {t.Fatal(err)}
    config.RuntimeParams["search_path"]=schema
    // Production uses pgx. In particular, rate-limit uniqueness must be
    // exercised with its PgError rather than the legacy lib/pq test driver.
    database:=stdlib.OpenDB(*config);database.SetMaxOpenConns(1);database.SetMaxIdleConns(1)
    t.Cleanup(func(){database.Close()})
    var admin,member,article,root,booking int64
    for i,target:=range []*int64{&admin,&member} {
        if err:=database.QueryRow(`INSERT INTO users(name,email,password_hash,is_admin) VALUES($1,$2,'fixture',$3) RETURNING id`,[]string{"Guide moderator","Guide learner"}[i],fmt.Sprintf("reading-%d@example.invalid",i),i==0).Scan(target);err!=nil {t.Fatal(err)}
        var block int64
        end:=[]string{"10:00","10:30"}[i]
        if err:=database.QueryRow(`INSERT INTO timetable(user_id,day_of_week,start_time,end_time,activity) VALUES($1,0,'09:00',$2,'Practice') RETURNING id`,*target,end).Scan(&block);err!=nil {t.Fatal(err)}
        if _,err:=database.Exec(`INSERT INTO activity_logs(user_id,timetable_id,log_date,completed,activity_name,summary) VALUES($1,$2,'2020-10-26',TRUE,'Practice',$3)`,*target,block,[]string{"ADMIN_ONLY_REFLECTION","MEMBER_ONLY_REFLECTION"}[i]);err!=nil {t.Fatal(err)}
    }
    if err:=database.QueryRow(`INSERT INTO learning_articles(slug,title,summary,body,category) VALUES('browser-guide','Plan a useful learning session','A practical guide to deliberate practice',$1,'Study habits') RETURNING id`,strings.Repeat("Read and try a useful learning step. ",40)).Scan(&article);err!=nil {t.Fatal(err)}
    if err:=database.QueryRow(`INSERT INTO learning_comments(article_id,user_id,body,minute_bucket) VALUES($1,$2,'What would you try first?',-1) RETURNING id`,article,admin).Scan(&root);err!=nil {t.Fatal(err)}
    if err:=database.QueryRow(`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,status,meeting_type) VALUES($1,$2,CURRENT_DATE-1,'09:00','10:00','Fixture record session','completed','online') RETURNING id`,admin,member).Scan(&booking);err!=nil {t.Fatal(err)}
    content,_:=json.Marshal(plannerTestCoaching())
    service:=plannerTestService(func(r *http.Request)(*http.Response,error){return plannerProviderResponse(200,string(content),"stop"),nil})
    router:=gin.New();router.Use(func(c *gin.Context){c.Set("db",database);c.Set("planner_ai_service",service);c.Next()})
    router.LoadHTMLGlob("../templates/*.html");router.Static("/static","../static")
    for path,name:=range map[string]string{"/learn":"learn","/learn/browser-guide":"article","/planner":"planner","/my-bookings":"my-bookings"}{template:=name;router.GET(path,func(c *gin.Context){c.HTML(200,template+".html",nil)})}
    router.GET("/api/v1/articles",ListLearningArticles);router.GET("/api/v1/articles/:slug",GetLearningArticle);router.GET("/api/v1/articles/:slug/comments",ListLearningComments)
    group:=router.Group("/api/v1",middleware.AuthRequired)
    group.GET("/me",func(c *gin.Context){var role bool;if err:=database.QueryRow(`SELECT is_admin FROM users WHERE id=$1`,c.GetInt64("user_id")).Scan(&role);err!=nil {c.JSON(500,gin.H{});return};c.JSON(200,gin.H{"id":c.GetInt64("user_id"),"is_admin":role})})
    group.POST("/articles/:slug/comments",AddLearningComment);group.GET("/planner/week",GetPlannerWeek);group.POST("/planner/analyze",AnalyzePlannerWeek);group.GET("/bookings",GetMyBookings)
    admins:=group.Group("/admin",middleware.AdminRequired);admins.DELETE("/articles/comments/:id",RemoveLearningComment)
    RegisterNotificationRoutes(group)
    server:=httptest.NewServer(router);defer server.Close()
    script,err:=filepath.Abs("../tests/ui/real-learning-coach.mjs");if err!=nil {t.Fatal(err)}
    command:=exec.Command("node",script);command.Env=append(os.Environ(),"L2E_TEST_ORIGIN="+server.URL,"L2E_ADMIN_TOKEN="+reflectionTestToken(t,admin),"L2E_MEMBER_TOKEN="+reflectionTestToken(t,member),fmt.Sprintf("L2E_ROOT_COMMENT=%d",root),fmt.Sprintf("L2E_RECORD_SESSION=%d",booking))
    if output,err:=command.CombinedOutput();err!=nil {t.Fatalf("learning coach browser: %v\n%s",err,output)}else{t.Log(string(output))}
    var comments int;if err:=database.QueryRow(`SELECT count(*) FROM learning_comments WHERE article_id=$1`,article).Scan(&comments);err!=nil||comments!=0 {t.Fatalf("moderation cascade: %d %v",comments,err)}
}
