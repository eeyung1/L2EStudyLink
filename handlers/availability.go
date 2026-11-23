package handlers

import (
    "database/sql"
    "context"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "sort"
    "net/http"
    "log"
    "time"

    "github.com/gin-gonic/gin"
)

type AvailabilitySlot struct {
    DayOfWeek int    `json:"day_of_week"`
    StartTime string `json:"start_time"`
    EndTime   string `json:"end_time"`
}

func SetAvailability(c *gin.Context) {
    userID := c.GetInt64("user_id")
    db := c.MustGet("db").(*sql.DB)

    var slots []AvailabilitySlot
    if err := c.ShouldBindJSON(&slots); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    for _, slot := range slots {
        start, startErr := time.Parse("15:04",slot.StartTime)
        end, endErr := time.Parse("15:04",slot.EndTime)
        if slot.DayOfWeek < 0 || slot.DayOfWeek > 6 || startErr != nil || endErr != nil || !start.Before(end) || len(slot.StartTime)!=5 || len(slot.EndTime)!=5 {
            c.JSON(http.StatusBadRequest,gin.H{"error":"Choose a valid day and start/end times to the minute; end must follow start"});return
        }
    }
    tx, err := db.BeginTx(c.Request.Context(),nil)
    if err != nil {c.JSON(500,gin.H{"error":"Failed to save availability"});return}
    defer tx.Rollback()
    c.Header("Cache-Control","private, no-store")
    var account int64
    if err:=tx.QueryRowContext(c.Request.Context(),"SELECT id FROM users WHERE id=$1 FOR UPDATE",userID).Scan(&account);err!=nil {c.JSON(500,gin.H{"error":"Could not lock your hours"});return}
    current,err:=availabilitySnapshot(c.Request.Context(),tx,userID)
    if err!=nil {c.JSON(500,gin.H{"error":"Could not read saved hours"});return}
    if match:=c.GetHeader("If-Match");match!=""&&match!=availabilityRevision(userID,current){c.JSON(412,gin.H{"error":"Your tutoring hours changed in another tab. Reload saved hours before trying again. Your form entries are kept."});return}

    _, err = tx.ExecContext(c.Request.Context(),"DELETE FROM availability WHERE user_id = $1", userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear availability"})
        return
    }

    // Insert new slots
    for _, slot := range slots {
        // Ensure time format is HH:MM:SS
        startTime := slot.StartTime
        endTime := slot.EndTime
        
        // Add seconds if missing
        if len(startTime) == 5 {
            startTime = startTime + ":00"
        }
        if len(endTime) == 5 {
            endTime = endTime + ":00"
        }
        
        _, err := tx.ExecContext(c.Request.Context(),`
            INSERT INTO availability (user_id, day_of_week, start_time, end_time) 
            VALUES ($1, $2, $3, $4)
        `, userID, slot.DayOfWeek, startTime, endTime)
        
        if err != nil {
            log.Printf("save availability for user %d: %v",userID,err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save your hours. Please try again."})
            return
        }
    }

    if err:=tx.Commit();err!=nil {c.JSON(500,gin.H{"error":"Failed to save availability"});return}
    c.JSON(http.StatusOK, gin.H{"message": "Availability saved successfully", "slots": slots, "revision":availabilityRevision(userID,slots)})
}

func GetAvailability(c *gin.Context) {
    c.Header("Cache-Control","private, no-store")
    userID:=c.GetInt64("user_id")
    slots,err:=availabilitySnapshot(c.Request.Context(),c.MustGet("db").(*sql.DB),userID)
    if err!=nil {log.Printf("read availability: %v",err);c.JSON(500,gin.H{"error":"Availability is temporarily unavailable"});return}
    revision:=availabilityRevision(userID,slots);c.Header("ETag",revision)
    if c.Query("revision")=="1" {c.JSON(200,gin.H{"slots":slots,"revision":revision});return}
    c.JSON(200,slots)
}

type availabilityReader interface {QueryContext(context.Context,string,...any)(*sql.Rows,error)}
func availabilitySnapshot(ctx context.Context,db availabilityReader,user int64)([]AvailabilitySlot,error){
    rows,err:=db.QueryContext(ctx,`SELECT day_of_week,CAST(start_time AS TEXT),CAST(end_time AS TEXT) FROM availability WHERE user_id=$1 ORDER BY day_of_week,start_time,end_time`,user)
    if err!=nil {return nil,err};defer rows.Close();slots:=[]AvailabilitySlot{}
    for rows.Next(){var slot AvailabilitySlot;var start,end string;if err:=rows.Scan(&slot.DayOfWeek,&start,&end);err!=nil{return nil,err};if len(start)<5||len(end)<5{return nil,fmt.Errorf("invalid stored time")};slot.StartTime=start[:5];slot.EndTime=end[:5];slots=append(slots,slot)};return slots,rows.Err()
}
func availabilityRevision(user int64,slots []AvailabilitySlot)string{
    ordered:=append([]AvailabilitySlot{},slots...);sort.Slice(ordered,func(i,j int)bool{a,b:=ordered[i],ordered[j];if a.DayOfWeek!=b.DayOfWeek{return a.DayOfWeek<b.DayOfWeek};if a.StartTime!=b.StartTime{return a.StartTime<b.StartTime};return a.EndTime<b.EndTime})
    value,_:=json.Marshal(ordered);return fmt.Sprintf("\"%x\"",sha256.Sum256([]byte(fmt.Sprint(user)+":"+string(value))))
}
