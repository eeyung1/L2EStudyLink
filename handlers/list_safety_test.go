package handlers

import (
 "encoding/json"
 "fmt"
 "net/http/httptest"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "sync"
 "testing"
 "github.com/gin-gonic/gin"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/stdlib"
 "L2EStudyLink/middleware"
)
func TestListSafetyFeatures(t *testing.T){
 t.Setenv("JWT_SECRET","list-safety-fixture-signing-secret");root:=timetableTestDB(t,false);var schema string;if err:=root.QueryRow(`SELECT current_schema()`).Scan(&schema);err!=nil {t.Fatal(err)}
 cfg,err:=pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"));if err!=nil {t.Fatal(err)};cfg.RuntimeParams["search_path"]=schema;cfg.DefaultQueryExecMode=pgx.QueryExecModeSimpleProtocol;db:=stdlib.OpenDB(*cfg);db.SetMaxOpenConns(2);db.SetMaxIdleConns(2);t.Cleanup(func(){db.Close()})
 ids:=[]int64{};for i:=0;i<45;i++{var id int64;if err:=db.QueryRow(`INSERT INTO users(name,email,password_hash,is_admin,is_suspended) VALUES($1,$2,'fixture',$3,FALSE) RETURNING id`,fmt.Sprintf("Account %02d",i),fmt.Sprintf("lists-%d@example.invalid",i),i==0).Scan(&id);err!=nil {t.Fatal(err)};ids=append(ids,id)}
 for i:=1;i<45;i++{if _,err:=db.Exec(`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,meeting_type,status) VALUES($1,$2,CURRENT_DATE+$3,'09:05','10:20','List practice','online',$4)`,ids[0],ids[i],i,[]string{"pending","confirmed"}[i%2]);err!=nil {t.Fatal(err)}}
 var firstArticle int64;for i:=0;i<45;i++{var id int64;if err:=db.QueryRow(`INSERT INTO learning_articles(slug,title,summary,body,category) VALUES($1,$2,'A useful learning exercise.',$3,'Practice') RETURNING id`,fmt.Sprintf("page-guide-%02d",i),fmt.Sprintf("Page guide %02d",i),strings.Repeat("Try one useful step. ",10)).Scan(&id);err!=nil {t.Fatal(err)};if i==0{firstArticle=id}}
 for i:=0;i<45;i++{if _,err:=db.Exec(`INSERT INTO learning_comments(article_id,user_id,body,minute_bucket) VALUES($1,$2,$3,$4)`,firstArticle,ids[0],fmt.Sprintf("Public discussion entry %02d",i),i);err!=nil {t.Fatal(err)}}
 r:=gin.New();r.Use(func(c *gin.Context){c.Set("db",db);c.Next()});r.LoadHTMLGlob("../templates/*.html");r.Static("/static","../static")
 for path,name:=range map[string]string{"/admin":"admin","/availability":"availability","/skills":"skills","/learn":"learn","/learn/:slug":"article"}{template:=name;r.GET(path,func(c *gin.Context){c.HTML(200,template+".html",nil)})}
 r.GET("/api/v1/articles",ListLearningArticles);r.GET("/api/v1/articles/:slug",GetLearningArticle);r.GET("/api/v1/articles/:slug/comments",ListLearningComments)
 g:=r.Group("/api/v1",middleware.AuthRequired);g.GET("/me",GetProfile);g.PUT("/profile",UpdateProfile);g.GET("/availability",GetAvailability);g.PUT("/availability",SetAvailability);g.GET("/booking-clock",GetBookingClock)
 a:=g.Group("/admin",middleware.AdminRequired);a.GET("/users",AdminUsers);a.GET("/bookings",AdminBookings);a.GET("/stats",AdminStats);a.GET("/product-email-audience",ProductEmailAudience)
 call:=func(method,path,body,match string,user int64)*httptest.ResponseRecorder{t.Helper();req:=httptest.NewRequest(method,path,strings.NewReader(body));req.Header.Set("Content-Type","application/json");if user>0{req.Header.Set("Authorization","Bearer "+reflectionTestToken(t,user))};if match!=""{req.Header.Set("If-Match",match)};w:=httptest.NewRecorder();r.ServeHTTP(w,req);return w}
 for _,check:=range []struct{path,key string;count int}{{"/api/v1/admin/users","items",45},{"/api/v1/admin/bookings","items",44},{"/api/v1/articles","articles",45},{"/api/v1/articles/page-guide-00/comments","comments",45}}{
  next:="";seen:=map[float64]bool{};for page:=0;page<10;page++{path:=check.path+"?paged=1&limit=7";if next!=""{path+="&after="+next};w:=call("GET",path,"","",ids[0]);if w.Code!=200{t.Fatalf("paged %s: %d %s",path,w.Code,w.Body.String())};var data map[string]any;if err:=json.Unmarshal(w.Body.Bytes(),&data);err!=nil{t.Fatal(err)};items:=data[check.key].([]any);if len(items)>7{t.Fatal("Page exceeded limit")};for _,raw:=range items{id:=raw.(map[string]any)["id"].(float64);if seen[id]{t.Fatalf("Repeated page entry in %s",path)};seen[id]=true};next=data["next_cursor"].(string);if !data["has_more"].(bool){break};if next==""{t.Fatal("Missing next cursor")}}
  if len(seen)!=check.count{t.Fatalf("%s returned %d of %d",check.path,len(seen),check.count)}
 }
 for _,path:=range []string{"/api/v1/admin/users?paged=1&q=Account%2044","/api/v1/admin/bookings?paged=1&q=Account%2044"}{w:=call("GET",path,"","",ids[0]);var data struct{Items []gin.H `json:"items"`};json.Unmarshal(w.Body.Bytes(),&data);if w.Code!=200||len(data.Items)!=1{t.Fatalf("Server search outside initial page: %s %s",path,w.Body.String())}}
 for _,path:=range []string{"/api/v1/admin/users?paged=1&limit=0","/api/v1/admin/bookings?paged=1&state=invalid","/api/v1/articles?paged=1&after=-1","/api/v1/articles/page-guide-00/comments?paged=1&limit=101"}{if w:=call("GET",path,"","",ids[0]);w.Code!=400{t.Fatalf("Invalid pagination accepted: %s %d",path,w.Code)}}
 if w:=call("GET","/api/v1/admin/users?paged=1","","",ids[1]);w.Code!=403{t.Fatal("Paged admin access crossed roles")}
 if w:=call("GET","/api/v1/admin/users","","",ids[0]);w.Code!=200||!strings.HasPrefix(w.Body.String(),"["){t.Fatal("Legacy array contract changed")}
 snapshot:=func()string{w:=call("GET","/api/v1/availability?revision=1","","",ids[0]);var d struct{Revision string `json:"revision"`};if w.Code!=200||json.Unmarshal(w.Body.Bytes(),&d)!=nil||d.Revision==""{t.Fatalf("Missing hours revision: %s",w.Body.String())};return d.Revision}
 old:=snapshot();one:=`[{"day_of_week":0,"start_time":"09:05","end_time":"10:20"}]`;if w:=call("PUT","/api/v1/availability",one,old,ids[0]);w.Code!=200{t.Fatalf("Initial conditional save: %s",w.Body.String())};if w:=call("PUT","/api/v1/availability","[]",old,ids[0]);w.Code!=412{t.Fatal("Stale tab overwrote saved hours")}
 revision:=snapshot();var wg sync.WaitGroup;codes:=make(chan int,2);for _,body:=range []string{`[{"day_of_week":2,"start_time":"13:00","end_time":"14:00"}]`,`[{"day_of_week":1,"start_time":"11:00","end_time":"12:00"}]`}{wg.Add(1);go func(body string){defer wg.Done();codes<-call("PUT","/api/v1/availability",body,revision,ids[0]).Code}(body)};wg.Wait();close(codes);success,conflict:=0,0;for code:=range codes{if code==200{success++}else if code==412{conflict++}else{t.Fatalf("Concurrent conditional save: %d",code)}};if success!=1||conflict!=1{t.Fatalf("Concurrent writes: %d saved, %d conflicted",success,conflict)}
 // Leave a predictable snapshot for the real browser's two-tab edit journey.
 if w:=call("PUT","/api/v1/availability",one,"",ids[0]);w.Code!=200{t.Fatal("Legacy availability write changed")}
 if os.Getenv("L2E_BROWSER_E2E")=="1"{server:=httptest.NewServer(r);defer server.Close();script,err:=filepath.Abs("../tests/ui/real-list-safety.mjs");if err!=nil{t.Fatal(err)};cmd:=exec.Command("node",script);cmd.Env=append(os.Environ(),"L2E_TEST_ORIGIN="+server.URL,"L2E_ADMIN_TOKEN="+reflectionTestToken(t,ids[0]));if output,err:=cmd.CombinedOutput();err!=nil{t.Fatalf("list safety browser: %v\n%s",err,output)}else{t.Log(string(output))}}
}
