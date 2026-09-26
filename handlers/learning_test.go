package handlers

import (
    "database/sql"
    "fmt"
    "net/http/httptest"
    "os"
    "strings"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"

    projectdb "L2EStudyLink/db"
)

func TestLearningRepliesStayInTheirArticle(t *testing.T) {
    dsn:=os.Getenv("TEST_DATABASE_URL")
    if dsn=="" {t.Skip("TEST_DATABASE_URL not configured")}
    database,err:=sql.Open("postgres",dsn)
    if err!=nil {t.Fatal(err)}
    defer database.Close()
    schema,err:=os.ReadFile("../schema.sql")
    if err!=nil {t.Fatal(err)}
    if _,err=database.Exec(string(schema)[:strings.Index(string(schema),"-- Skills table")]);err!=nil {t.Fatal(err)}
    t.Setenv("DATABASE_URL",dsn)
    if err=projectdb.InitDB();err!=nil {t.Fatal(err)}
    defer projectdb.CloseDB()
    stamp:=time.Now().UnixNano()
    slug:=fmt.Sprintf("reply-test-%d",stamp)
    other:=fmt.Sprintf("other-reply-test-%d",stamp)
    var rootAuthor,replier int64
    for i,id:=range []*int64{&rootAuthor,&replier} {
        if err=database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES($1,$2,'test') RETURNING id`,fmt.Sprintf("Test fellow %d",i),fmt.Sprintf("reply-test-%d-%d@example.com",stamp,i)).Scan(id);err!=nil {t.Fatal(err)}
    }
    defer database.Exec(`DELETE FROM users WHERE id IN ($1,$2)`,rootAuthor,replier)
    for _,s:=range []string{slug,other} {
        if _,err=database.Exec(`INSERT INTO learning_articles(slug,title,summary,body) VALUES($1,'Test guide','Test summary','Test article text')`,s);err!=nil {t.Fatal(err)}
    }
    defer database.Exec(`DELETE FROM learning_articles WHERE slug IN ($1,$2)`,slug,other)
    var parent int64
    if err=database.QueryRow(`INSERT INTO learning_comments(article_id,user_id,body,minute_bucket) SELECT id,$2,'A first question',$3 FROM learning_articles WHERE slug=$1 RETURNING id`,slug,rootAuthor,stamp).Scan(&parent);err!=nil {t.Fatal(err)}

    router:=gin.New()
    router.Use(func(c *gin.Context){c.Set("db",database);c.Set("user_id",replier);c.Next()})
    router.POST("/articles/:slug/comments",AddLearningComment)
    post:=func(slug,body string) int {
        recorder:=httptest.NewRecorder();request:=httptest.NewRequest("POST","/articles/"+slug+"/comments",strings.NewReader(body));request.Header.Set("Content-Type","application/json");router.ServeHTTP(recorder,request);return recorder.Code
    }
    payload:=fmt.Sprintf(`{"body":"A helpful reply","parent_id":%d}`,parent)
    if code:=post(other,payload);code!=404 {t.Fatalf("cross-article reply: %d",code)}
    if code:=post(slug,payload);code!=201 {t.Fatalf("same-article reply: %d",code)}
    var replyParent int64
    if err=database.QueryRow(`SELECT id FROM learning_comments WHERE parent_id=$1 AND user_id=$2`,parent,replier).Scan(&replyParent);err!=nil {t.Fatal(err)}
    if code:=post(slug,fmt.Sprintf(`{"body":"Nested reply","parent_id":%d}`,replyParent));code!=404 {t.Fatalf("nested reply: %d",code)}
    var canInsert bool
    if err=database.QueryRow(`SELECT CASE WHEN EXISTS (SELECT 1 FROM pg_roles WHERE rolname='anon') THEN has_table_privilege('anon','learning_comments','INSERT') ELSE false END`).Scan(&canInsert);err!=nil||canInsert {t.Fatalf("public Data API grants: %v, %v",canInsert,err)}
}
