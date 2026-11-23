package handlers

import (
 "fmt"
 "net/http/httptest"
 "os"
 "os/exec"
 "path/filepath"
 "testing"
 "github.com/gin-gonic/gin"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/stdlib"
 "L2EStudyLink/middleware"
)

func TestPublicAccountBrowserJourney(t *testing.T) {
 if os.Getenv("L2E_BROWSER_E2E")!="1" {t.Skip("Browser journey runs in UI CI")}
 t.Setenv("JWT_SECRET","public-account-browser-fixture-secret")
 fixture:=timetableTestDB(t,false);var schema string;if err:=fixture.QueryRow(`SELECT current_schema()`).Scan(&schema);err!=nil {t.Fatal(err)}
 cfg,err:=pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"));if err!=nil {t.Fatal(err)};cfg.RuntimeParams["search_path"]=schema
 db:=stdlib.OpenDB(*cfg);db.SetMaxOpenConns(1);db.SetMaxIdleConns(1);t.Cleanup(func(){db.Close()})
 var admin,peer int64
 for i,target:=range []*int64{&admin,&peer}{if err:=db.QueryRow(`INSERT INTO users(name,email,password_hash,is_admin) VALUES($1,$2,'fixture',$3) RETURNING id`,[]string{"Fixture admin","Fixture peer"}[i],fmt.Sprintf("account-fixture-%d@example.invalid",i),i==0).Scan(target);err!=nil {t.Fatal(err)}}
 r:=gin.New();r.Use(func(c *gin.Context){c.Set("db",db);c.Next()});r.LoadHTMLGlob("../templates/*.html");r.Static("/static","../static")
 for path,name:=range map[string]string{"/":"landing","/page/signup":"signup","/page/login":"login","/page/forgot-password":"forgot-password","/page/reset-password":"reset-password","/learn":"learn","/learn/agentic-workflow":"article","/skills":"skills","/availability":"availability","/admin":"admin"}{template:=name;r.GET(path,func(c *gin.Context){c.HTML(200,template+".html",nil)})}
 r.POST("/api/v1/signup",Signup);r.POST("/api/v1/login",Login);r.POST("/api/v1/reset-password",ResetPassword)
 g:=r.Group("/api/v1",middleware.AuthRequired);g.GET("/me",GetProfile);g.PUT("/profile",UpdateProfile);g.PUT("/me/product-emails",SetProductEmails);g.POST("/skills",AddSkill);g.PUT("/skills/:skill",EditSkill);g.DELETE("/skills/:skill",RemoveSkill);g.GET("/availability",GetAvailability);g.PUT("/availability",SetAvailability)
 a:=g.Group("/admin",middleware.AdminRequired);a.GET("/stats",AdminStats);a.GET("/users",AdminUsers);a.GET("/bookings",AdminBookings);a.PUT("/users/:id/suspend",AdminToggleSuspend)
 // Test-only reset seeding on this isolated server; no transactional email is sent.
 r.POST("/__fixture/reset",func(c *gin.Context){var id int64;if err:=db.QueryRow(`SELECT id FROM users WHERE email='browser-new@example.invalid'`).Scan(&id);err!=nil {t.Error(err);c.Status(500);return};token:="otp:0:fixture-nonce:"+resetCodeHash(id,"fixture-nonce","123456");if _,err:=db.Exec(`INSERT INTO password_reset_tokens(user_id,token,expires_at) VALUES($1,$2,NOW()+INTERVAL '10 minutes')`,id,token);err!=nil {t.Error(err);c.Status(500);return};c.Status(204)})
 server:=httptest.NewServer(r);defer server.Close();script,err:=filepath.Abs("../tests/ui/real-public-account.mjs");if err!=nil {t.Fatal(err)};cmd:=exec.Command("node",script);cmd.Env=append(os.Environ(),"L2E_TEST_ORIGIN="+server.URL,"L2E_ADMIN_TOKEN="+reflectionTestToken(t,admin),fmt.Sprintf("L2E_PEER_ID=%d",peer))
 if output,err:=cmd.CombinedOutput();err!=nil {t.Fatalf("public account journey: %v\n%s",err,output)}else{t.Log(string(output))}
 var opted bool;var bio string;if err:=db.QueryRow(`SELECT marketing_opt_in_at IS NOT NULL,bio FROM users WHERE email='browser-new@example.invalid'`).Scan(&opted,&bio);err!=nil||opted||bio!="My private learning bio" {t.Fatalf("saved settings: %v %q %v",opted,bio,err)}
}
