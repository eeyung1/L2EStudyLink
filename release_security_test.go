package main

import (
 "database/sql"
 "fmt"
 "net/http/httptest"
 "os"
 "strings"
 "testing"
 "time"
 "github.com/gin-gonic/gin"
 "github.com/golang-jwt/jwt/v5"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/stdlib"
 _ "github.com/lib/pq"
)

// Exercise the production router against a disposable schema, never existing tables.
func TestReleaseAuthorizationAndPublicBoundary(t *testing.T) {
 dsn:=os.Getenv("TEST_DATABASE_URL");if dsn=="" {t.Skip("TEST_DATABASE_URL not configured")};t.Setenv("JWT_SECRET","release-audit-fixture-signing-secret")
 root,err:=sql.Open("postgres",dsn);if err!=nil {t.Fatal(err)};defer root.Close();schema:=fmt.Sprintf("release_audit_%d",time.Now().UnixNano());if _,err=root.Exec(`CREATE SCHEMA `+schema);err!=nil {t.Fatal(err)};defer func(){if _,err:=root.Exec(`DROP SCHEMA `+schema+` CASCADE`);err!=nil {t.Error(err)}}()
 root.SetMaxOpenConns(1);if _,err=root.Exec(`SET search_path TO `+schema);err!=nil {t.Fatal(err)};source,err:=os.ReadFile("schema.sql");if err!=nil {t.Fatal(err)};if _,err=root.Exec(string(source));err!=nil {t.Fatal(err)}
 cfg,err:=pgx.ParseConfig(dsn);if err!=nil {t.Fatal(err)};cfg.RuntimeParams["search_path"]=schema;db:=stdlib.OpenDB(*cfg);defer db.Close();db.SetMaxOpenConns(1)
 var admin,member int64;for i,target:=range []*int64{&admin,&member}{if err:=db.QueryRow(`INSERT INTO users(name,email,password_hash,is_admin,public_handle,portfolio_public) VALUES($1,$2,'HASH_MUST_STAY_PRIVATE',$3,$4,TRUE) RETURNING id`,[]string{"Audit admin","Audit member"}[i],fmt.Sprintf("audit-%d@example.invalid",i),i==0,[]string{"audit_admin","audit_member"}[i]).Scan(target);err!=nil {t.Fatal(err)}}
 const canary="REFLECTION_MUST_STAY_PRIVATE";const audio="/uploads/1-00000000-0000-4000-8000-000000000001.wav"
 if _,err=db.Exec(`INSERT INTO activity_logs(user_id,log_date,completed,activity_name,summary,audio_url) VALUES($1,CURRENT_DATE,TRUE,'Private study',$2,$3)`,member,canary,audio);err!=nil {t.Fatal(err)}
 if _,err=db.Exec(`INSERT INTO learning_articles(slug,title,summary,body,category) VALUES('audit-guide','Public audit guide','An intentionally public guide.','Explicitly public editorial text, unrelated to private reflections.','Practice')`);err!=nil {t.Fatal(err)}
 token:=func(id int64)string{v,err:=jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.MapClaims{"user_id":id,"exp":time.Now().Add(time.Hour).Unix()}).SignedString([]byte(os.Getenv("JWT_SECRET")));if err!=nil {t.Fatal(err)};return v}
 gin.SetMode(gin.TestMode);r:=newRouter(db)
 request:=func(method,path,body,bearer string)*httptest.ResponseRecorder{t.Helper();req:=httptest.NewRequest(method,path,strings.NewReader(body));req.Header.Set("Content-Type","application/json");if bearer!="" {req.Header.Set("Authorization","Bearer "+bearer)};w:=httptest.NewRecorder();r.ServeHTTP(w,req);return w}
 routes:=[]struct{method,path string}{{"GET","/api/v1/admin/stats"},{"GET","/api/v1/admin/users"},{"GET","/api/v1/admin/bookings"},{"GET","/api/v1/admin/product-email-audience"},{"GET","/api/v1/admin/product-email-audience.csv"},{"DELETE",fmt.Sprintf("/api/v1/admin/users/%d",member)},{"PUT",fmt.Sprintf("/api/v1/admin/users/%d/suspend",member)},{"POST","/api/v1/admin/articles"},{"DELETE","/api/v1/admin/articles/comments/99999"}}
 // Ensure newly added admin endpoints cannot silently escape this complete matrix.
 covered:=map[string]bool{};for _,check:=range routes{covered[check.method+" "+check.path]=true}
 count:=0;for _,route:=range r.Routes(){if strings.HasPrefix(route.Path,"/api/v1/admin/"){count++;path:=route.Path;if strings.Contains(path,"/comments/"){path=strings.ReplaceAll(path,":id","99999")}else{path=strings.ReplaceAll(path,":id",fmt.Sprint(member))};if !covered[route.Method+" "+path]{t.Fatalf("Unaudited admin route: %s %s",route.Method,route.Path)}}};if count!=len(covered){t.Fatalf("Admin route inventory changed: %d routes, %d audited",count,len(covered))}
 for _,check:=range routes{if check.method=="GET"{w:=request(check.method,check.path,"",token(admin));if w.Code!=200||!strings.Contains(w.Header().Get("Cache-Control"),"no-store"){t.Fatalf("Active admin read %s: %d cache=%s",check.path,w.Code,w.Header().Get("Cache-Control"))}}}
 for _,state:=range []string{"guest","member","suspended admin","demoted admin","deleted admin"}{
  bearer:="";if state=="member" {bearer=token(member)};if strings.Contains(state,"admin"){bearer=token(admin)}
  switch state {case "suspended admin":_,err=db.Exec(`UPDATE users SET is_suspended=TRUE WHERE id=$1`,admin);case "demoted admin":_,err=db.Exec(`UPDATE users SET is_suspended=FALSE,is_admin=FALSE WHERE id=$1`,admin);case "deleted admin":_,err=db.Exec(`DELETE FROM users WHERE id=$1`,admin)};if err!=nil {t.Fatal(err)}
  for _,check:=range routes{want:=403;if state=="guest"{want=401};w:=request(check.method,check.path,`{}`,bearer);if w.Code!=want {t.Fatalf("%s %s %s: %d %s",state,check.method,check.path,w.Code,w.Body.String())}}
 }
 for _,path:=range []string{"/","/learn","/learn/audit-guide","/api/v1/articles","/api/v1/articles/audit-guide","/api/v1/articles/audit-guide/comments","/api/v1/search","/api/v1/search-with-availability",fmt.Sprintf("/api/v1/tutors/%d",member),"/api/v1/portfolio/audit_member"}{w:=request("GET",path,"","");if w.Code!=200 {t.Fatalf("public %s: %d %s",path,w.Code,w.Body.String())};for _,private:=range []string{canary,audio,"HASH_MUST_STAY_PRIVATE"}{if strings.Contains(w.Body.String(),private){t.Fatalf("Private field leaked via %s",path)}}}
 own:=request("GET","/api/v1/reflections","",token(member));if own.Code!=200||!strings.Contains(own.Body.String(),canary){t.Fatal("Private canary not readable by its owner")}
 var peer int64;if err:=db.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Audit peer','audit-peer@example.invalid','fixture') RETURNING id`).Scan(&peer);err!=nil {t.Fatal(err)}
 other:=request("GET","/api/v1/reflections","",token(peer));if other.Code!=200||strings.Contains(other.Body.String(),canary){t.Fatal("Reflection crossed accounts")}
 if _,err:=db.Exec(`UPDATE users SET portfolio_public=FALSE WHERE id=$1`,member);err!=nil {t.Fatal(err)};if w:=request("GET","/api/v1/portfolio/audit_member","","");w.Code!=404 {t.Fatalf("Revoked portfolio stayed public: %d",w.Code)}
 for _,path:=range []string{"/api/v1/me","/api/v1/reflections","/api/v1/planner/week",audio}{if w:=request("GET",path,"","");w.Code!=401 {t.Fatalf("Private guest route %s: %d",path,w.Code)}}
 for _,name:=range []string{"does-not-exist","app-navigation","layout"}{if w:=request("GET","/page/"+name,"","");w.Code!=404 {t.Fatalf("Unknown/partial page %s: %d",name,w.Code)}}
}
