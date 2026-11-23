package handlers

import (
 "fmt"
 "strconv"
 "strings"
 "time"
 "github.com/gin-gonic/gin"
 "database/sql"
)

type listPage struct { Limit int; After int64; Query,State string }
func readListPage(c *gin.Context,states string)(listPage,bool){
 p:=listPage{Limit:20,Query:strings.TrimSpace(c.Query("q")),State:c.DefaultQuery("state","all")};var err error
 if value:=c.Query("limit");value!="" {p.Limit,err=strconv.Atoi(value);if err!=nil||p.Limit<1||p.Limit>100 {c.JSON(400,gin.H{"error":"Page size must be 1–100"});return p,false}}
 if value:=c.Query("after");value!="" {p.After,err=strconv.ParseInt(value,10,64);if err!=nil||p.After<=0 {c.JSON(400,gin.H{"error":"Invalid page cursor"});return p,false}}
 validState:=false;for _,value:=range strings.Split(states,"|"){if p.State==value{validState=true}}
 if len(p.Query)>120||!validState{c.JSON(400,gin.H{"error":"Invalid search or status filter"});return p,false};return p,true
}
func sendListPage(c *gin.Context,key string,items []gin.H,p listPage){
 more:=len(items)>p.Limit;if more {items=items[:p.Limit]};next:="";if more {next=fmt.Sprint(items[len(items)-1]["id"])}
 c.JSON(200,gin.H{key:items,"has_more":more,"next_cursor":next})
}
func pagedAdminUsers(c *gin.Context){
 p,ok:=readListPage(c,"all|active|suspended|admin");if !ok{return};db:=c.MustGet("db").(*sql.DB)
 rows,err:=db.QueryContext(c.Request.Context(),`SELECT id,name,email,rating,total_sessions,no_show_count,is_suspended,is_admin FROM users WHERE id>$1 AND ($2='' OR strpos(lower(name||' '||email||' '||id::text),lower($2))>0) AND ($3='all' OR ($3='active' AND NOT is_suspended) OR ($3='suspended' AND is_suspended) OR ($3='admin' AND is_admin)) ORDER BY id LIMIT $4`,p.After,p.Query,p.State,p.Limit+1)
 if err!=nil {c.JSON(500,gin.H{"error":"Could not load accounts"});return};defer rows.Close();items:=[]gin.H{}
 for rows.Next(){var id int64;var name,email string;var rating float64;var sessions,missed int;var suspended,admin bool;if err:=rows.Scan(&id,&name,&email,&rating,&sessions,&missed,&suspended,&admin);err!=nil {c.JSON(500,gin.H{"error":"Could not read accounts"});return};items=append(items,gin.H{"id":id,"name":name,"email":email,"rating":rating,"total_sessions":sessions,"no_show_count":missed,"is_suspended":suspended,"is_admin":admin})};if rows.Err()!=nil {c.JSON(500,gin.H{"error":"Could not load accounts"});return};sendListPage(c,"items",items,p)
}
func pagedAdminBookings(c *gin.Context){
 p,ok:=readListPage(c,"all|pending|confirmed|cancelled|completed|no_show");if !ok{return};db:=c.MustGet("db").(*sql.DB)
 rows,err:=db.QueryContext(c.Request.Context(),`SELECT b.id,t.name,s.name,b.session_date,b.start_time,b.status FROM bookings b JOIN users t ON t.id=b.tutor_id JOIN users s ON s.id=b.student_id WHERE ($1::bigint=0 OR b.id<$1) AND ($2='' OR strpos(lower(t.name||' '||s.name||' '||b.id::text),lower($2))>0) AND ($3='all' OR b.status=$3) ORDER BY b.id DESC LIMIT $4`,p.After,p.Query,p.State,p.Limit+1)
 if err!=nil {c.JSON(500,gin.H{"error":"Could not load bookings"});return};defer rows.Close();items:=[]gin.H{}
 for rows.Next(){var id int64;var tutor,student,date,start,status string;if err:=rows.Scan(&id,&tutor,&student,&date,&start,&status);err!=nil {c.JSON(500,gin.H{"error":"Could not read bookings"});return};items=append(items,gin.H{"id":id,"tutor":tutor,"student":student,"date":date,"time":start,"status":status})};if rows.Err()!=nil {c.JSON(500,gin.H{"error":"Could not load bookings"});return};sendListPage(c,"items",items,p)
}
func pagedLearningArticles(c *gin.Context){
 p,ok:=readListPage(c,"all");if !ok{return};category:=c.Query("category");if len(category)>40 {c.JSON(400,gin.H{"error":"Invalid category"});return};db:=c.MustGet("db").(*sql.DB)
 rows,err:=db.QueryContext(c.Request.Context(),`SELECT id,slug,title,summary,category,published_at FROM learning_articles WHERE ($1::bigint=0 OR id<$1) AND ($2='' OR strpos(lower(title||' '||summary||' '||category),lower($2))>0) AND ($3='' OR category=$3) ORDER BY id DESC LIMIT $4`,p.After,p.Query,category,p.Limit+1)
 if err!=nil {c.JSON(500,gin.H{"error":"Could not load guides"});return};items:=[]gin.H{}
 for rows.Next(){var id int64;var slug,title,summary,category string;var published time.Time;if err:=rows.Scan(&id,&slug,&title,&summary,&category,&published);err!=nil {rows.Close();c.JSON(500,gin.H{"error":"Could not read guides"});return};items=append(items,gin.H{"id":id,"slug":slug,"title":title,"summary":summary,"category":category,"published_at":published})};err=rows.Err();rows.Close();if err!=nil {c.JSON(500,gin.H{"error":"Could not load guides"});return}
 categories:=[]string{};cats,err:=db.QueryContext(c.Request.Context(),`SELECT DISTINCT category FROM learning_articles ORDER BY category`);if err!=nil {c.JSON(500,gin.H{"error":"Could not load guide categories"});return};defer cats.Close();for cats.Next(){var v string;if err:=cats.Scan(&v);err!=nil {c.JSON(500,gin.H{"error":"Could not read categories"});return};categories=append(categories,v)};if cats.Err()!=nil {c.JSON(500,gin.H{"error":"Could not load categories"});return}
 more:=len(items)>p.Limit;if more {items=items[:p.Limit]};next:="";if more {next=fmt.Sprint(items[len(items)-1]["id"])};c.JSON(200,gin.H{"articles":items,"categories":categories,"has_more":more,"next_cursor":next})
}
func pagedLearningComments(c *gin.Context){
 p,ok:=readListPage(c,"all");if !ok{return};db:=c.MustGet("db").(*sql.DB);var article int64;err:=db.QueryRowContext(c.Request.Context(),`SELECT id FROM learning_articles WHERE slug=$1`,c.Param("slug")).Scan(&article);if err==sql.ErrNoRows {c.JSON(404,gin.H{"error":"Article not found"});return};if err!=nil {c.JSON(500,gin.H{"error":"Could not load discussion"});return}
 rows,err:=db.QueryContext(c.Request.Context(),`SELECT lc.id,u.name,lc.body,lc.created_at,lc.parent_id FROM learning_comments lc JOIN users u ON u.id=lc.user_id WHERE lc.article_id=$1 AND ($2::bigint=0 OR lc.id<$2) ORDER BY lc.id DESC LIMIT $3`,article,p.After,p.Limit+1);if err!=nil {c.JSON(500,gin.H{"error":"Could not load discussion"});return};defer rows.Close();items:=[]gin.H{}
 for rows.Next(){var id int64;var name,body string;var created time.Time;var parent sql.NullInt64;if err:=rows.Scan(&id,&name,&body,&created,&parent);err!=nil {c.JSON(500,gin.H{"error":"Could not read discussion"});return};var parentID any;if parent.Valid {parentID=parent.Int64};items=append(items,gin.H{"id":id,"name":name,"body":body,"created_at":created,"parent_id":parentID})};if rows.Err()!=nil {c.JSON(500,gin.H{"error":"Could not load discussion"});return};sendListPage(c,"comments",items,p)
}
