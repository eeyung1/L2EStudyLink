package handlers

import (
 "database/sql"
 "fmt"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/gin-gonic/gin"
)

func testBookingOutcomeAndReview(t *testing.T, db *sql.DB, tutor,student,other int64) {
 var past,future int64
 for _,v:=range []struct{id *int64; date string}{{&past,"CURRENT_DATE - 2"},{&future,"CURRENT_DATE + 2"}} {
  q:=`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,meeting_type,status) VALUES($1,$2,`+v.date+`,'09:00','10:00','Go testing','online','confirmed') RETURNING id`
  if err:=db.QueryRow(q,tutor,student).Scan(v.id);err!=nil {t.Fatal(err)}
 }
 r:=gin.New();r.Use(func(c *gin.Context){var id int64;fmt.Sscan(c.GetHeader("X-Test-User"),&id);c.Set("user_id",id);c.Set("db",db);c.Next()})
 r.PUT("/bookings/:id/outcome",CompleteBooking);r.POST("/bookings/:id/reviews",CreateReview);r.GET("/bookings",GetMyBookings)
 call:=func(who int64,method,path,body string) int {t.Helper();req:=httptest.NewRequest(method,path,strings.NewReader(body));req.Header.Set("Content-Type","application/json");req.Header.Set("X-Test-User",fmt.Sprint(who));w:=httptest.NewRecorder();r.ServeHTTP(w,req);if w.Code>=500 {t.Fatalf("%s %s: %d %s",method,path,w.Code,w.Body.String())};return w.Code}
 pastPath:=fmt.Sprintf("/bookings/%d",past);futurePath:=fmt.Sprintf("/bookings/%d",future)
 if n:=call(tutor,"PUT",futurePath+"/outcome",`{"status":"completed"}`);n!=409 {t.Fatalf("future outcome: %d",n)}
 if n:=call(other,"PUT",pastPath+"/outcome",`{"status":"completed"}`);n!=403 {t.Fatalf("stranger outcome: %d",n)}
 if n:=call(student,"POST",pastPath+"/reviews",`{"rating":5,"comment":"Helpful guidance"}`);n!=409 {t.Fatalf("premature review: %d",n)}
 if n:=call(tutor,"PUT",pastPath+"/outcome",`{"status":"completed"}`);n!=200 {t.Fatalf("completed: %d",n)}
 if n:=call(tutor,"PUT",pastPath+"/outcome",`{"status":"no_show"}`);n!=409 {t.Fatalf("repeated outcome: %d",n)}
 if n:=call(tutor,"POST",pastPath+"/reviews",`{"rating":5,"comment":"Helpful guidance"}`);n!=403 {t.Fatalf("tutor self review: %d",n)}
 if n:=call(student,"POST",pastPath+"/reviews",`{"rating":5,"comment":"Helpful guidance"}`);n!=201 {t.Fatalf("student review: %d",n)}
 if n:=call(student,"POST",pastPath+"/reviews",`{"rating":2,"comment":"Another review"}`);n!=409 {t.Fatalf("duplicate review: %d",n)}
 var count int;var rating float64
 if err:=db.QueryRow(`SELECT total_reviews,rating FROM users WHERE id=$1`,tutor).Scan(&count,&rating);err!=nil||count!=1||rating!=5 {t.Fatalf("rating: %d %.2f %v",count,rating,err)}
 var sessions int
 if err:=db.QueryRow(`SELECT total_sessions FROM users WHERE id=$1`,student).Scan(&sessions);err!=nil||sessions!=1 {t.Fatalf("sessions: %d %v",sessions,err)}
 // A second past booking records a no-show and cannot receive a review.
 var missed int64
 if err:=db.QueryRow(`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,meeting_type,status) VALUES($1,$2,CURRENT_DATE-3,'11:00','12:00','Go practice','online','confirmed') RETURNING id`,tutor,student).Scan(&missed);err!=nil {t.Fatal(err)}
 missedPath:=fmt.Sprintf("/bookings/%d",missed)
 if n:=call(tutor,"PUT",missedPath+"/outcome",`{"status":"no_show"}`);n!=200 {t.Fatalf("no-show: %d",n)}
 if n:=call(student,"POST",missedPath+"/reviews",`{"rating":5,"comment":"Helpful guidance"}`);n!=409 {t.Fatalf("no-show review: %d",n)}
 if err:=db.QueryRow(`SELECT no_show_count FROM users WHERE id=$1`,student).Scan(&count);err!=nil||count!=1 {t.Fatalf("no-show count: %d %v",count,err)}
}
