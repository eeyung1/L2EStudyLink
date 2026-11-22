package handlers

import (
    "fmt"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "github.com/gin-gonic/gin"
    "L2EStudyLink/middleware"
)

func TestBookingBrowserJourney(t *testing.T){
    if os.Getenv("L2E_BROWSER_E2E")!="1" {t.Skip("Browser journey runs in the UI CI job")}
    t.Setenv("JWT_SECRET","booking-browser-fixture-secret");t.Setenv("DISCORD_WEBHOOK_URL","");t.Setenv("RESEND_API_KEY","")
    database:=timetableTestDB(t,false);tutor,_:=timetableFixture(t,database);var student int64
    if _,err:=database.Exec(`UPDATE users SET name='Go mentor',bio='Learn API validation together' WHERE id=$1`,tutor);err!=nil {t.Fatal(err)}
    if err:=database.QueryRow(`INSERT INTO users(name,email,password_hash) VALUES('Booking learner','real-booking@example.invalid','fixture') RETURNING id`).Scan(&student);err!=nil {t.Fatal(err)}
    var date string;var day int;if err:=database.QueryRow(`SELECT to_char(CURRENT_DATE+2,'YYYY-MM-DD'),EXTRACT(ISODOW FROM CURRENT_DATE+2)::int-1`).Scan(&date,&day);err!=nil {t.Fatal(err)}
    if _,err:=database.Exec(`INSERT INTO skills(user_id,skill_name,proficiency) VALUES($1,'Go','advanced')`,tutor);err!=nil {t.Fatal(err)}
    if _,err:=database.Exec(`INSERT INTO availability(user_id,day_of_week,start_time,end_time) VALUES($1,$2,'09:00','11:00')`,tutor,day);err!=nil {t.Fatal(err)}
    router:=gin.New();router.Use(func(c *gin.Context){c.Set("db",database);c.Next()});router.LoadHTMLGlob("../templates/*.html");router.Static("/static","../static")
    for _,name:=range []string{"search-page","my-bookings"}{page:=name;router.GET("/"+page,func(c *gin.Context){c.HTML(200,page+".html",nil)})}
    router.GET("/api/v1/search-with-availability",SearchTutorsWithAvailability);router.GET("/api/v1/booking-clock",GetBookingClock)
    group:=router.Group("/api/v1",middleware.AuthRequired)
    group.GET("/me",func(c *gin.Context){c.JSON(200,gin.H{"id":c.GetInt64("user_id"),"name":"Browser fixture"})})
    group.GET("/bookings",GetMyBookings);group.POST("/bookings",CreateBooking);group.PUT("/bookings/:id/status",UpdateBookingStatus);group.DELETE("/bookings/:id",CancelBooking)
    server:=httptest.NewServer(router);defer server.Close()
    script,err:=filepath.Abs("../tests/ui/real-bookings.mjs");if err!=nil {t.Fatal(err)};command:=exec.Command("node",script)
    command.Env=append(os.Environ(),"L2E_TEST_ORIGIN="+server.URL,"L2E_BOOKING_DATE="+date,"L2E_STUDENT_TOKEN="+reflectionTestToken(t,student),"L2E_TUTOR_TOKEN="+reflectionTestToken(t,tutor))
    if result,err:=command.CombinedOutput();err!=nil {t.Fatalf("real booking browser journey: %v\n%s",err,result)}else{t.Log(string(result))}
    var count int;if err:=database.QueryRow(`SELECT count(*) FROM notifications WHERE event_key LIKE 'booking:%'`).Scan(&count);err!=nil||count!=6 {t.Fatalf("booking notification lifecycle: %d %v",count,err)}
    var status string;if err:=database.QueryRow(`SELECT status FROM bookings WHERE student_id=$1`,student).Scan(&status);err!=nil||status!="cancelled" {t.Fatal(fmt.Sprintf("final status: %q %v",status,err))}
}
