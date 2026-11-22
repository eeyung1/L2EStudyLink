package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http/httptest"
    "net/url"
    "os"
    "strings"
    "sync"
    "testing"
    "github.com/gin-gonic/gin"
)

func bookingTestRouter(database *sql.DB,who int64)*gin.Engine {
    router:=gin.New();router.Use(func(c *gin.Context){c.Set("db",database);c.Set("user_id",who);c.Next()})
    router.POST("/bookings",CreateBooking);router.GET("/bookings",GetMyBookings);router.GET("/clock",GetBookingClock);router.PUT("/bookings/:id/status",UpdateBookingStatus);router.DELETE("/bookings/:id",CancelBooking)
    return router
}
func bookingCall(router *gin.Engine,method,path,body string)*httptest.ResponseRecorder {request:=httptest.NewRequest(method,path,strings.NewReader(body));request.Header.Set("Content-Type","application/json");out:=httptest.NewRecorder();router.ServeHTTP(out,request);return out}
func TestBookingValidationAndConcurrentConflicts(t *testing.T){
    t.Setenv("DISCORD_WEBHOOK_URL","");t.Setenv("RESEND_API_KEY","")
    database:=timetableTestDB(t,false);tutor,_:=timetableFixture(t,database)
    var student,other int64
    for i,target:=range []*int64{&student,&other}{if err:=database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Booking fixture',$1,'fixture') RETURNING id`,fmt.Sprintf("booking-%d@example.invalid",i)).Scan(target);err!=nil {t.Fatal(err)}}
    var date,schema string;var day int
    if err:=database.QueryRow(`SELECT to_char(CURRENT_DATE+2,'YYYY-MM-DD'),EXTRACT(ISODOW FROM CURRENT_DATE+2)::int-1,current_schema()`).Scan(&date,&day,&schema);err!=nil {t.Fatal(err)}
    if _,err:=database.Exec(`INSERT INTO availability(user_id,day_of_week,start_time,end_time) VALUES($1,$2,'09:00','11:00'),($1,$2,'10:00','12:00'),($1,$2,'13:00','14:00')`,tutor,day);err!=nil {t.Fatal(err)}
    body:=func(id int64,start,end,topic string)string {value,_:=json.Marshal(map[string]any{"tutor_id":id,"date":date,"start_time":start,"end_time":end,"topic":topic,"meeting_type":"online"});return string(value)}
    router:=bookingTestRouter(database,student)
    if out:=bookingCall(bookingTestRouter(database,tutor),"POST","/bookings",body(tutor,"09:00","11:00","Self"));out.Code!=400 {t.Fatalf("self booking: %d",out.Code)}
    for _,invalid:=range []string{body(tutor,"08:00","09:00","Unpublished"),body(tutor,"09:00","11:00"," "),body(tutor,"12:00","09:00","Bad time"),strings.Replace(body(tutor,"09:00","11:00","Past"),date,"2020-01-01",1)} {
        if out:=bookingCall(router,"POST","/bookings",invalid);out.Code!=400 {t.Fatalf("invalid request: %d %s",out.Code,out.Body.String())}
    }
    if _,err:=database.Exec(`UPDATE users SET is_suspended=TRUE WHERE id=$1`,tutor);err!=nil {t.Fatal(err)}
    if out:=bookingCall(router,"POST","/bookings",body(tutor,"09:00","11:00","Suspended"));out.Code!=400 {t.Fatalf("suspended tutor: %d",out.Code)}
    database.Exec(`UPDATE users SET is_suspended=FALSE WHERE id=$1`,tutor)
    // Each connection uses the same isolated test schema, allowing real concurrent transactions.
    parsed,err:=url.Parse(os.Getenv("TEST_DATABASE_URL"));if err!=nil {t.Fatal(err)};query:=parsed.Query();query.Set("search_path",schema);parsed.RawQuery=query.Encode()
    second,err:=sql.Open("postgres",parsed.String());if err!=nil {t.Fatal(err)};defer second.Close();second.SetMaxOpenConns(1)
    var group sync.WaitGroup;codes:=make(chan int,2)
    for _,r:=range []*gin.Engine{router,bookingTestRouter(second,other)} {group.Add(1);go func(r *gin.Engine){defer group.Done();codes<-bookingCall(r,"POST","/bookings",body(tutor,"09:00","11:00","Concurrent request")).Code}(r)}
    group.Wait();close(codes);created,conflict:=0,0;for code:=range codes {if code==201 {created++};if code==409 {conflict++}}
    if created!=1||conflict!=1 {t.Fatalf("concurrent slot: %d created, %d rejected",created,conflict)}
    if out:=bookingCall(router,"POST","/bookings",body(tutor,"10:00","12:00","Overlapping"));out.Code!=409 {t.Fatalf("overlap: %d %s",out.Code,out.Body.String())}
    if out:=bookingCall(router,"POST","/bookings",body(tutor,"13:00:00","14:00:00","Separate slot"));out.Code!=201 {t.Fatalf("valid second slot: %d %s",out.Code,out.Body.String())}
    var count int;if err:=database.QueryRow(`SELECT count(*) FROM notifications WHERE event_key LIKE 'booking:%:pending'`).Scan(&count);err!=nil||count!=4 {t.Fatalf("durable creation notifications: %d %v",count,err)}
    var id int64;if err:=database.QueryRow(`SELECT id FROM bookings WHERE topic='Separate slot'`).Scan(&id);err!=nil {t.Fatal(err)}
    if out:=bookingCall(bookingTestRouter(database,tutor),"PUT",fmt.Sprintf("/bookings/%d/status",id),`{"status":"confirmed"}`);out.Code!=200 {t.Fatalf("accept: %d",out.Code)}
    if out:=bookingCall(router,"DELETE",fmt.Sprintf("/bookings/%d",id),"");out.Code!=200 {t.Fatalf("student cancel confirmed: %d",out.Code)}
    if err:=database.QueryRow(`SELECT count(*) FROM notifications WHERE event_key LIKE $1`,fmt.Sprintf("booking:%d:%%",id)).Scan(&count);err!=nil||count!=6 {t.Fatalf("transition notifications: %d %v",count,err)}
}

func TestBookingActionFlags(t *testing.T){
    database:=timetableTestDB(t,false);tutor,_:=timetableFixture(t,database);var student int64
    if err:=database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Learner','action-flags@example.invalid','fixture') RETURNING id`).Scan(&student);err!=nil {t.Fatal(err)}
    for _,state:=range []string{"pending","confirmed","cancelled","completed","no_show"}{for _,offset:=range []int{-2,2}{if _,err:=database.Exec(`INSERT INTO bookings(tutor_id,student_id,session_date,start_time,end_time,topic,meeting_type,status) VALUES($1,$2,CURRENT_DATE+$3::int,'09:00','10:00','State fixture','online',$4)`,tutor,student,offset,state);err!=nil {t.Fatal(err)}}}
    for _,who:=range []int64{tutor,student}{out:=bookingCall(bookingTestRouter(database,who),"GET","/bookings","");if out.Code!=200 {t.Fatal(out.Body.String())};var values []struct{Status,Role,Timezone string;CanAccept bool `json:"can_accept"`;CanCancel bool `json:"can_cancel"`;CanComplete bool `json:"can_complete"`;CanReview bool `json:"can_review"`};if err:=json.Unmarshal(out.Body.Bytes(),&values);err!=nil {t.Fatal(err)};if len(values)!=10 {t.Fatalf("state fixtures: %d",len(values))};for _,b:=range values{if b.Timezone=="" {t.Fatal("missing booking clock")};if b.CanAccept&&(b.Role!="tutor"||b.Status!="pending") {t.Fatal("invalid accept action")};if b.CanComplete&&(b.Role!="tutor"||b.Status!="confirmed") {t.Fatal("invalid completion action")};if b.CanReview&&(b.Role!="student"||b.Status!="completed") {t.Fatal("invalid review action")};if b.CanCancel&&b.Status!="pending"&&b.Status!="confirmed" {t.Fatal("invalid cancel action")}}}
}
