package handlers

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync/atomic"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/stdlib"
    "L2EStudyLink/focus"
    "L2EStudyLink/middleware"
)

func TestFocusIdentityBrowserJourney(t *testing.T){
    if os.Getenv("L2E_BROWSER_E2E")!="1" {t.Skip("Browser journey runs in UI CI")}
    t.Setenv("JWT_SECRET","focus-identity-browser-fixture-secret");t.Setenv("GROQ_API_KEY","fixture-no-network")
    fixture:=timetableTestDB(t,false);var schema string;if err:=fixture.QueryRow(`SELECT current_schema()`).Scan(&schema);err!=nil {t.Fatal(err)}
    cfg,err:=pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"));if err!=nil {t.Fatal(err)};cfg.RuntimeParams["search_path"]=schema
    database:=stdlib.OpenDB(*cfg);database.SetMaxOpenConns(1);database.SetMaxIdleConns(1);t.Cleanup(func(){database.Close()})
    var a,b int64
    for i,target:=range []*int64{&a,&b}{
        if err:=database.QueryRow(`INSERT INTO users(name,email,password_hash,bio) VALUES($1,$2,'fixture','A learner-shared bio') RETURNING id`,[]string{"Learner Alpha","Learner Beta"}[i],fmt.Sprintf("focus-identity-%d@example.invalid",i)).Scan(target);err!=nil {t.Fatal(err)}
        var block int64;if err:=database.QueryRow(`INSERT INTO timetable(user_id,day_of_week,start_time,end_time,activity) VALUES($1,0,'09:00',$2,$3) RETURNING id`,*target,[]string{"10:00","10:30"}[i],[]string{"Alpha practice","Beta practice"}[i]).Scan(&block);err!=nil {t.Fatal(err)}
        if _,err:=database.Exec(`INSERT INTO activity_logs(user_id,timetable_id,log_date,completed,activity_name,summary) VALUES($1,$2,'2020-10-26',TRUE,'Practice',$3),($1,NULL,'2020-10-19',TRUE,'Past','OUTSIDE_WEEK_CANARY'),($1,NULL,'2020-10-27',FALSE,'Incomplete','INCOMPLETE_CANARY')`,*target,block,[]string{"ALPHA_ONLY_CANARY ","BETA_ONLY_CANARY "}[i]+strings.Repeat("x",1100));err!=nil {t.Fatal(err)}
        if _,err:=database.Exec(`INSERT INTO skills(user_id,skill_name,proficiency) VALUES($1,$2,'intermediate')`,*target,[]string{"Go","Design"}[i]);err!=nil {t.Fatal(err)}
    }
    var providerFail atomic.Bool;var calls atomic.Int64;valid,_:=json.Marshal(plannerTestCoaching())
    service:=plannerTestService(func(req *http.Request)(*http.Response,error){
        body,_:=io.ReadAll(req.Body);var request struct{Messages []struct{Role,Content string} `json:"messages"`};if err:=json.Unmarshal(body,&request);err!=nil {t.Error(err)}
        if len(request.Messages)!=2||request.Messages[0].Role!="system"||request.Messages[0].Content!=plannerCoachPrompt {t.Error("Unexpected coaching prompt source")}
        user:=request.Messages[1].Content;alpha:=strings.Contains(user,"ALPHA_ONLY_CANARY");beta:=strings.Contains(user,"BETA_ONLY_CANARY")
        if alpha==beta||strings.Contains(user,"OUTSIDE_WEEK_CANARY")||strings.Contains(user,"INCOMPLETE_CANARY"){t.Error("Coaching context crossed account, week or completion boundaries")}
        calls.Add(1);if providerFail.Swap(false){return plannerProviderResponse(400,"fixture unavailable","stop"),nil};return plannerProviderResponse(200,string(valid),"stop"),nil
    })
    for _,id:=range []int64{a,b}{plannerReviewQuota.Lock();delete(plannerReviewQuota.Entries,id);plannerReviewQuota.Unlock()}
    t.Cleanup(func(){plannerReviewQuota.Lock();defer plannerReviewQuota.Unlock();delete(plannerReviewQuota.Entries,a);delete(plannerReviewQuota.Entries,b)})
    var advance atomic.Int64;epoch:=time.Now();rooms,err:=focus.NewStore(focus.Config{Clock:func()time.Time{return epoch.Add(time.Duration(advance.Load())*time.Second)},PresenceTTL:2*time.Hour,IdleTTL:3*time.Hour});if err!=nil {t.Fatal(err)}
    fs:=newFocusService(rooms);fs.tick=50*time.Millisecond;fs.heartbeat=100*time.Millisecond
    router:=gin.New();router.Use(func(c *gin.Context){c.Set("db",database);c.Set("planner_ai_service",service);c.Next()});router.LoadHTMLGlob("../templates/*.html");router.Static("/static","../static")
    for path,name:=range map[string]string{"/focus":"focus","/planner":"planner","/portfolio":"portfolio","/u/:handle":"portfolio","/timetable":"timetable"}{template:=name;router.GET(path,func(c *gin.Context){c.HTML(200,template+".html",nil)})}
    router.GET("/api/v1/portfolio/:handle",GetPublicPortfolio)
    group:=router.Group("/api/v1",middleware.AuthRequired)
    group.GET("/me",func(c *gin.Context){id:=c.GetInt64("user_id");name:="Learner Alpha";if id==b{name="Learner Beta"};c.JSON(200,gin.H{"id":id,"name":name})})
    group.GET("/planner/week",GetPlannerWeek);group.POST("/planner/analyze",AnalyzePlannerWeek)
    group.GET("/portfolio/me",GetMyPortfolio);group.PUT("/portfolio/handle",ClaimPortfolioHandle);group.PUT("/portfolio/visibility",SetPortfolioVisibility)
    group.GET("/timetable",GetTimetable);group.GET("/reflections",GetReflections);group.POST("/reflections",AddReflection);group.GET("/reflections/retention",GetReflectionRetention)
    fs.register(group)
    // Test-local controls only, on the isolated httptest server; never app routes.
    router.POST("/__fixture/break",func(c *gin.Context){advance.Store(1501);c.Status(204)})
    router.POST("/__fixture/provider-failure",func(c *gin.Context){providerFail.Store(true);c.Status(204)})
    server:=httptest.NewServer(router);defer server.Close()
    script,err:=filepath.Abs("../tests/ui/real-focus-identity.mjs");if err!=nil {t.Fatal(err)};command:=exec.Command("node",script)
    command.Env=append(os.Environ(),"L2E_TEST_ORIGIN="+server.URL,"L2E_ALPHA_TOKEN="+reflectionTestToken(t,a),"L2E_BETA_TOKEN="+reflectionTestToken(t,b))
    if output,err:=command.CombinedOutput();err!=nil {t.Fatalf("focus identity browser: %v\n%s",err,output)}else{t.Log(string(output))}
    if calls.Load()!=4 {t.Fatalf("Expected three Alpha attempts and one Beta attempt, got %d",calls.Load())}
    var count int;if err:=database.QueryRow(`SELECT count(*) FROM activity_logs WHERE summary='Explicit reflection after focus.' AND user_id=$1`,b).Scan(&count);err!=nil||count!=1 {t.Fatalf("explicit handoff save: %d %v",count,err)}
}
