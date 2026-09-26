package handlers

import (
    "database/sql"
    "errors"
    "net/http"
    "regexp"
    "strconv"
    "strings"
    "time"
    "unicode/utf8"

    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgconn"
)

var articleSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ListLearningArticles(c *gin.Context) {
    db := c.MustGet("db").(*sql.DB)
    rows, err := db.QueryContext(c.Request.Context(), `SELECT slug,title,summary,category,published_at FROM learning_articles ORDER BY published_at DESC,id DESC LIMIT 40`)
    if err != nil { c.JSON(500, gin.H{"error":"Could not load articles"}); return }
    defer rows.Close()
    items := make([]gin.H, 0)
    for rows.Next() {
        var slug,title,summary,category string
        var published time.Time
        if err = rows.Scan(&slug,&title,&summary,&category,&published); err != nil { c.JSON(500, gin.H{"error":"Could not load articles"}); return }
        items=append(items,gin.H{"slug":slug,"title":title,"summary":summary,"category":category,"published_at":published})
    }
    if rows.Err()!=nil {c.JSON(500,gin.H{"error":"Could not load articles"});return}
    c.JSON(200,gin.H{"articles":items})
}

func GetLearningArticle(c *gin.Context) {
    db:=c.MustGet("db").(*sql.DB)
    var id int64
    var title,summary,body,category string
    var published time.Time
    err:=db.QueryRowContext(c.Request.Context(),`SELECT id,title,summary,body,category,published_at FROM learning_articles WHERE slug=$1`,c.Param("slug")).Scan(&id,&title,&summary,&body,&category,&published)
    if errors.Is(err,sql.ErrNoRows) {c.JSON(404,gin.H{"error":"Article not found"});return}
    if err!=nil {c.JSON(500,gin.H{"error":"Could not load article"});return}
    c.JSON(200,gin.H{"id":id,"slug":c.Param("slug"),"title":title,"summary":summary,"body":body,"category":category,"published_at":published})
}

func ListLearningComments(c *gin.Context) {
    db:=c.MustGet("db").(*sql.DB)
    var articleID int64
    err:=db.QueryRowContext(c.Request.Context(),`SELECT id FROM learning_articles WHERE slug=$1`,c.Param("slug")).Scan(&articleID)
    if errors.Is(err,sql.ErrNoRows) {c.JSON(404,gin.H{"error":"Article not found"});return}
    if err!=nil {c.JSON(500,gin.H{"error":"Could not load comments"});return}
    rows,err:=db.QueryContext(c.Request.Context(),`SELECT lc.id,u.name,lc.body,lc.created_at,lc.parent_id FROM learning_comments lc JOIN users u ON u.id=lc.user_id WHERE lc.article_id=$1 ORDER BY lc.created_at DESC,lc.id DESC LIMIT 200`,articleID)
    if err!=nil {c.JSON(500,gin.H{"error":"Could not load comments"});return}
    defer rows.Close()
    comments:=make([]gin.H,0)
    for rows.Next() {
        var id int64; var name,body string; var created time.Time; var parent sql.NullInt64
        if err=rows.Scan(&id,&name,&body,&created,&parent);err!=nil {c.JSON(500,gin.H{"error":"Could not load comments"});return}
        var parentID any
        if parent.Valid {parentID=parent.Int64}
        comments=append(comments,gin.H{"id":id,"name":name,"body":body,"created_at":created,"parent_id":parentID})
    }
    if rows.Err()!=nil {c.JSON(500,gin.H{"error":"Could not load comments"});return}
    c.JSON(200,gin.H{"comments":comments})
}

func AddLearningComment(c *gin.Context) {
    var input struct { Body string `json:"body"`; ParentID *int64 `json:"parent_id"` }
    if c.ShouldBindJSON(&input)!=nil {c.JSON(400,gin.H{"error":"Write a comment or question"});return}
    input.Body=strings.TrimSpace(input.Body)
    length:=utf8.RuneCountInString(input.Body)
    if length<3||length>1200 {c.JSON(400,gin.H{"error":"Use 3 to 1200 characters"});return}
    if input.ParentID!=nil && *input.ParentID<1 {c.JSON(400,gin.H{"error":"Invalid reply target"});return}
    db:=c.MustGet("db").(*sql.DB)
    var id int64
    err:=db.QueryRowContext(c.Request.Context(),`INSERT INTO learning_comments(article_id,user_id,body,parent_id,minute_bucket)
        SELECT a.id,$2,$3,$4,FLOOR(EXTRACT(EPOCH FROM NOW())/60)::bigint
        FROM learning_articles a JOIN users u ON u.id=$2 AND NOT COALESCE(u.is_suspended,FALSE)
        LEFT JOIN learning_comments p ON p.id=$4 AND p.article_id=a.id AND p.parent_id IS NULL
        WHERE a.slug=$1 AND ($4::bigint IS NULL OR p.id IS NOT NULL) RETURNING id`,c.Param("slug"),c.GetInt64("user_id"),input.Body,input.ParentID).Scan(&id)
    if errors.Is(err,sql.ErrNoRows) {c.JSON(404,gin.H{"error":"Article or comment not found, or commenting is unavailable"});return}
    var pgErr *pgconn.PgError
    if errors.As(err,&pgErr)&&pgErr.Code=="23505" {c.JSON(429,gin.H{"error":"Please wait a minute before posting again"});return}
    if err!=nil {c.JSON(500,gin.H{"error":"Could not post comment"});return}
    c.JSON(201,gin.H{"id":id,"parent_id":input.ParentID,"message":"Posted publicly"})
}

func PublishLearningArticle(c *gin.Context) {
    var input struct { Slug string `json:"slug"`; Title string `json:"title"`; Summary string `json:"summary"`; Category string `json:"category"`; Body string `json:"body"` }
    if c.ShouldBindJSON(&input)!=nil {c.JSON(400,gin.H{"error":"Invalid article"});return}
    input.Title=strings.TrimSpace(input.Title);input.Summary=strings.TrimSpace(input.Summary);input.Body=strings.TrimSpace(input.Body);input.Category=strings.TrimSpace(input.Category)
    if !articleSlug.MatchString(input.Slug)||len(input.Slug)>120||utf8.RuneCountInString(input.Title)<5||utf8.RuneCountInString(input.Title)>180||utf8.RuneCountInString(input.Summary)<10||utf8.RuneCountInString(input.Summary)>300||utf8.RuneCountInString(input.Category)<2||utf8.RuneCountInString(input.Category)>40||utf8.RuneCountInString(input.Body)<80||utf8.RuneCountInString(input.Body)>20000 {c.JSON(400,gin.H{"error":"Check article fields and lengths"});return}
    db:=c.MustGet("db").(*sql.DB)
    _,err:=db.ExecContext(c.Request.Context(),`INSERT INTO learning_articles(slug,title,summary,category,body) VALUES($1,$2,$3,$4,$5)`,input.Slug,input.Title,input.Summary,input.Category,input.Body)
    var pgErr *pgconn.PgError
    if errors.As(err,&pgErr)&&pgErr.Code=="23505" {c.JSON(409,gin.H{"error":"Article slug already exists"});return}
    if err!=nil {c.JSON(500,gin.H{"error":"Could not publish article"});return}
    c.JSON(201,gin.H{"slug":input.Slug})
}

func RemoveLearningComment(c *gin.Context) {
    id,err:=strconv.ParseInt(c.Param("id"),10,64)
    if err!=nil||id<1 {c.JSON(400,gin.H{"error":"Invalid comment"});return}
    db:=c.MustGet("db").(*sql.DB)
    result,err:=db.ExecContext(c.Request.Context(),`DELETE FROM learning_comments WHERE id=$1`,id)
    if err!=nil {c.JSON(500,gin.H{"error":"Could not remove comment"});return}
    count,_:=result.RowsAffected()
    if count==0 {c.JSON(404,gin.H{"error":"Comment not found"});return}
    c.JSON(http.StatusOK,gin.H{"message":"Comment removed"})
}
