package handlers

import (
 "database/sql"
 "errors"
 "net/http"
 "strconv"
 "strings"

 "github.com/gin-gonic/gin"
 "github.com/jackc/pgx/v5/pgconn"
)

// CompleteBooking allows the tutor to record the outcome after a confirmed
// session has ended. The conditional update makes repeated decisions safe.
func CompleteBooking(c *gin.Context) {
 var input struct { Status string `json:"status" binding:"required,oneof=completed no_show"` }
 if err:=c.ShouldBindJSON(&input);err!=nil { c.JSON(400,gin.H{"error":"Choose completed or no_show"});return }
 id,err:=strconv.ParseInt(c.Param("id"),10,64)
 if err!=nil||id<1 {c.JSON(400,gin.H{"error":"Invalid booking"});return}
 db:=c.MustGet("db").(*sql.DB)
 tx,err:=db.BeginTx(c.Request.Context(),nil)
 if err!=nil {c.JSON(500,gin.H{"error":"Could not record outcome"});return}
 defer tx.Rollback()
 var tutor,student int64
 var status string
 var ended bool
 err=tx.QueryRowContext(c.Request.Context(),`SELECT tutor_id,student_id,status,(session_date + end_time <= NOW()) FROM bookings WHERE id=$1 FOR UPDATE`,id).Scan(&tutor,&student,&status,&ended)
 if errors.Is(err,sql.ErrNoRows){c.JSON(404,gin.H{"error":"Booking not found"});return}
 if err!=nil {c.JSON(500,gin.H{"error":"Could not load booking"});return}
 if tutor!=c.GetInt64("user_id"){c.JSON(403,gin.H{"error":"Only the tutor can record the outcome"});return}
 if status!="confirmed" {c.JSON(409,gin.H{"error":"Only confirmed sessions can receive an outcome"});return}
 if !ended {c.JSON(409,gin.H{"error":"Wait until the session ends"});return}
 if _,err=tx.ExecContext(c.Request.Context(),`UPDATE bookings SET status=$1 WHERE id=$2`,input.Status,id);err!=nil {c.JSON(500,gin.H{"error":"Could not record outcome"});return}
 if input.Status=="completed" {
  _,err=tx.ExecContext(c.Request.Context(),`UPDATE users SET total_sessions=COALESCE(total_sessions,0)+1 WHERE id IN ($1,$2)`,tutor,student)
 } else {
  _,err=tx.ExecContext(c.Request.Context(),`UPDATE users SET no_show_count=COALESCE(no_show_count,0)+1 WHERE id=$1`,student)
 }
 if err!=nil {c.JSON(500,gin.H{"error":"Could not record outcome"});return}
 if err=tx.Commit();err!=nil {c.JSON(500,gin.H{"error":"Could not record outcome"});return}
 c.JSON(200,gin.H{"status":input.Status})
}

func CreateReview(c *gin.Context) {
 id,err:=strconv.ParseInt(c.Param("id"),10,64)
 if err!=nil||id<1 {c.JSON(400,gin.H{"error":"Invalid booking"});return}
 var input struct { Rating int `json:"rating"`; Comment string `json:"comment"`; WouldBookAgain *bool `json:"would_book_again"` }
 if err=c.ShouldBindJSON(&input);err!=nil {c.JSON(400,gin.H{"error":"Invalid review"});return}
 input.Comment=strings.TrimSpace(input.Comment)
 if input.Rating<1||input.Rating>5||len(input.Comment)<10||len(input.Comment)>1000 {c.JSON(400,gin.H{"error":"Choose 1–5 stars and write 10–1000 characters"});return}
 db:=c.MustGet("db").(*sql.DB)
 tx,err:=db.BeginTx(c.Request.Context(),nil)
 if err!=nil {c.JSON(500,gin.H{"error":"Could not save review"});return}
 defer tx.Rollback()
 var tutor,student int64
 var status string
 err=tx.QueryRowContext(c.Request.Context(),`SELECT tutor_id,student_id,status FROM bookings WHERE id=$1 FOR UPDATE`,id).Scan(&tutor,&student,&status)
 if errors.Is(err,sql.ErrNoRows){c.JSON(404,gin.H{"error":"Booking not found"});return}
 if err!=nil {c.JSON(500,gin.H{"error":"Could not load booking"});return}
 if student!=c.GetInt64("user_id"){c.JSON(403,gin.H{"error":"Only the student can review this session"});return}
 if status!="completed" {c.JSON(409,gin.H{"error":"Only completed sessions can be reviewed"});return}
 var again any
 if input.WouldBookAgain!=nil {again=*input.WouldBookAgain}
 _,err=tx.ExecContext(c.Request.Context(),`INSERT INTO reviews(booking_id,reviewer_id,reviewee_id,rating,comment,would_book_again) VALUES($1,$2,$3,$4,$5,$6)`,id,student,tutor,input.Rating,input.Comment,again)
 if err!=nil {var pgErr *pgconn.PgError;if errors.As(err,&pgErr)&&pgErr.Code=="23505" {c.JSON(409,gin.H{"error":"You already reviewed this session"});return};c.JSON(500,gin.H{"error":"Could not save review"});return}
 _,err=tx.ExecContext(c.Request.Context(),`UPDATE users SET rating=(SELECT ROUND(AVG(r.rating)::numeric,2) FROM reviews r WHERE r.reviewee_id=$1),total_reviews=(SELECT COUNT(*) FROM reviews r WHERE r.reviewee_id=$1) WHERE id=$1`,tutor)
 if err!=nil {c.JSON(500,gin.H{"error":"Could not save review"});return}
 if err=tx.Commit();err!=nil {c.JSON(500,gin.H{"error":"Could not save review"});return}
 c.JSON(201,gin.H{"message":"Review saved"})
}
