package handlers

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "mime/multipart"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"
    "L2EStudyLink/middleware"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

func reflectionTestToken(t *testing.T,id int64) string {
    t.Helper()
    value,err:=jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.MapClaims{"user_id":id,"exp":time.Now().Add(time.Hour).Unix()}).SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err!=nil {t.Fatal(err)};return value
}
func privateReflectionRouter(database *sql.DB) *gin.Engine {
    router:=gin.New();router.Use(func(c *gin.Context){c.Set("db",database);c.Next()})
    RegisterVoiceRoutes(router)
    group:=router.Group("/api/v1",middleware.AuthRequired)
    group.GET("/reflections",GetReflections);group.POST("/reflections",AddReflection)
    group.GET("/timetable",GetTimetable);group.POST("/timetable",AddTimeBlock);group.DELETE("/timetable/:id",DeleteTimeBlock)
    group.GET("/reflections/retention",GetReflectionRetention)
    return router
}
func TestReflectionMediaPrivacy(t *testing.T) {
    t.Setenv("JWT_SECRET","reflection-privacy-test-secret")
    t.Setenv("VOICE_UPLOAD_DIR",t.TempDir())
    database:=timetableTestDB(t,false);owner,block:=timetableFixture(t,database)
    var other,otherBlock int64
    if err:=database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Other','privacy-other@example.invalid','fixture') RETURNING id`).Scan(&other);err!=nil {t.Fatal(err)}
    if err:=database.QueryRow(`INSERT INTO timetable(user_id,day_of_week,start_time,end_time,activity) VALUES($1,2,'10:00','11:00','Other block') RETURNING id`,other).Scan(&otherBlock);err!=nil {t.Fatal(err)}
    router:=privateReflectionRouter(database)
    request:=func(method,path,body string,user int64)*httptest.ResponseRecorder {
        t.Helper();req:=httptest.NewRequest(method,path,strings.NewReader(body));req.Header.Set("Content-Type","application/json")
        if user>0 {req.Header.Set("Authorization","Bearer "+reflectionTestToken(t,user))}
        out:=httptest.NewRecorder();router.ServeHTTP(out,req);return out
    }
    upload:=func(path string,user int64)*httptest.ResponseRecorder {
        var body bytes.Buffer;form:=multipart.NewWriter(&body);file,_:=form.CreateFormFile("audio","fixture.wav");file.Write([]byte("RIFF fixture audio"));form.Close()
        req:=httptest.NewRequest("POST",path,&body);req.Header.Set("Content-Type",form.FormDataContentType());if user>0 {req.Header.Set("Authorization","Bearer "+reflectionTestToken(t,user))}
        out:=httptest.NewRecorder();router.ServeHTTP(out,req);return out
    }
    for _,path:=range []string{"/api/v1/voice/upload","/api/voice/upload"} {if got:=upload(path,0);got.Code!=401 {t.Fatalf("anonymous upload %s: %d",path,got.Code)}}
    result:=upload("/api/v1/voice/upload",owner);if result.Code!=200 {t.Fatalf("upload: %d %s",result.Code,result.Body.String())}
    var attachment VoiceResponse;if err:=json.Unmarshal(result.Body.Bytes(),&attachment);err!=nil {t.Fatal(err)}
    for _,item:=range []struct{user int64;code int}{{0,401},{owner,200},{other,404}} {
        out:=request("GET",attachment.AudioURL,"",item.user)
        if out.Code!=item.code {t.Fatalf("audio user %d: %d",item.user,out.Code)}
        if item.user==owner && out.Header().Get("Cache-Control")!="private, no-store" {t.Fatal("private media was cacheable")}
    }
    rangeReq:=httptest.NewRequest("GET",attachment.AudioURL,nil);rangeReq.Header.Set("Authorization","Bearer "+reflectionTestToken(t,owner));rangeReq.Header.Set("Range","bytes=0-3");out:=httptest.NewRecorder();router.ServeHTTP(out,rangeReq)
    if out.Code!=206||out.Body.String()!="RIFF" {t.Fatalf("audio seeking: %d %q",out.Code,out.Body.String())}
    payload:=func(b int64,audio,summary,date string) string {value,_:=json.Marshal(map[string]any{"timetable_id":b,"log_date":date,"summary":summary,"audio_url":audio});return string(value)}
    if got:=request("POST","/api/v1/reflections",payload(otherBlock,attachment.AudioURL,"Other tries attaching","2026-11-23"),other);got.Code!=400 {t.Fatalf("foreign audio attached: %d",got.Code)}
    for _,bad:=range []string{"https://example.invalid/track.wav","/uploads/../../secret","/uploads/999-00000000-0000-4000-8000-000000000001.wav"} {
        if got:=request("POST","/api/v1/reflections",payload(block,bad,"Summary","2026-11-23"),owner);got.Code!=400 {t.Fatalf("unsafe attachment accepted: %d",got.Code)}
    }
    for _,bad:=range []string{payload(block,"","  ","2026-11-23"),payload(block,"","Valid","bad-date"),payload(block,"",strings.Repeat("a",2001),"2026-11-23")} {
        if got:=request("POST","/api/v1/reflections",bad,owner);got.Code!=400 {t.Fatalf("invalid reflection accepted: %d",got.Code)}
    }
    if got:=request("POST","/api/v1/reflections",payload(block,attachment.AudioURL,"Private reflection","2026-11-23"),owner);got.Code!=201 {t.Fatalf("save: %d %s",got.Code,got.Body.String())}
    if got:=request("GET","/api/v1/reflections","",other);strings.Contains(got.Body.String(),"Private reflection")||strings.Contains(got.Body.String(),attachment.AudioURL) {t.Fatal("foreign history leaked")}
    if got:=request("GET","/api/v1/reflections","",owner);got.Header().Get("Cache-Control")!="private, no-store" {t.Fatal("history response was cacheable")}
    legacy:="00000000-0000-4000-8000-000000000002.wav"
    if err:=os.WriteFile(filepath.Join(voiceDirectory(),legacy),[]byte("legacy audio"),0600);err!=nil {t.Fatal(err)}
    if _,err:=database.Exec(`UPDATE activity_logs SET audio_url=$1 WHERE user_id=$2`,"/uploads/"+legacy,owner);err!=nil {t.Fatal(err)}
    if got:=request("GET","/uploads/"+legacy,"",owner);got.Code!=200 {t.Fatalf("owner legacy audio: %d",got.Code)}
    if got:=request("GET","/uploads/"+legacy,"",other);got.Code!=404 {t.Fatalf("foreign legacy audio: %d",got.Code)}
    if got:=request("GET","/uploads/00000000-0000-4000-8000-000000000003.wav","",owner);got.Code!=404 {t.Fatalf("unowned orphan audio: %d",got.Code)}
    if got:=request("DELETE",fmt.Sprintf("/api/v1/timetable/%d",block),"",owner);got.Code!=200 {t.Fatalf("safe delete: %d %s",got.Code,got.Body.String())}
    if got:=request("GET","/uploads/"+legacy,"",owner);got.Code!=200 {t.Fatalf("detached history audio lost: %d",got.Code)}
    // Files belong to the uploader; assigning a foreign user_id in JSON cannot bypass this.
}

func TestUnsafeRetentionBlocksDeletion(t *testing.T) {
    database:=timetableTestDB(t,true);owner,block:=timetableFixture(t,database)
    router:=gin.New();router.Use(func(c *gin.Context){c.Set("db",database);c.Set("user_id",owner);c.Next()})
    router.GET("/retention",GetReflectionRetention);router.DELETE("/timetable/:id",DeleteTimeBlock)
    for _,method:=range []string{"GET","DELETE"} {
        path:="/retention";if method=="DELETE" {path=fmt.Sprintf("/timetable/%d",block)}
        out:=httptest.NewRecorder();router.ServeHTTP(out,httptest.NewRequest(method,path,nil));if out.Code!=503 {t.Fatalf("unverified retention allowed %s: %d",method,out.Code)}
    }
    var count int;if err:=database.QueryRow(`SELECT count(*) FROM timetable WHERE id=$1`,block).Scan(&count);err!=nil||count!=1 {t.Fatal("unsafe delete changed block")}
}

func TestReflectionRowLevelPrivacy(t *testing.T) {
    database:=timetableTestDB(t,false);owner,block:=timetableFixture(t,database)
    if _,err:=database.Exec(`INSERT INTO activity_logs(user_id,timetable_id,log_date,summary) VALUES($1,$2,'2026-11-23','Private RLS fixture')`,owner,block);err!=nil {t.Fatal(err)}
    role:=fmt.Sprintf("reflection_reader_%d",time.Now().UnixNano())
    if _,err:=database.Exec("CREATE ROLE "+role+" NOLOGIN");err!=nil {t.Fatal(err)}
    defer database.Exec("DROP ROLE "+role)
    tx,err:=database.Begin();if err!=nil {t.Fatal(err)};defer tx.Rollback()
    var schema string;if err:=tx.QueryRow(`SELECT current_schema()`).Scan(&schema);err!=nil {t.Fatal(err)}
    if _,err:=tx.Exec("GRANT USAGE ON SCHEMA "+schema+" TO "+role+"; GRANT SELECT ON activity_logs TO "+role+"; SET LOCAL ROLE "+role);err!=nil {t.Fatal(err)}
    var count int;if err:=tx.QueryRow(`SELECT count(*) FROM activity_logs`).Scan(&count);err!=nil||count!=0 {t.Fatalf("unprivileged role saw private reflections: %d %v",count,err)}
}
