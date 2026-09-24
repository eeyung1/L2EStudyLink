package handlers

import (
    "database/sql"
    "encoding/csv"
    "log"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

// Audience excludes suspended accounts and fellows who did not explicitly opt in.
const productEmailAudience = `FROM users WHERE is_suspended=FALSE AND marketing_opt_in_at IS NOT NULL`

func ProductEmailAudience(c *gin.Context) {
    c.Header("Cache-Control", "no-store")
    db := c.MustGet("db").(*sql.DB)
    var eligible int
    if err := db.QueryRowContext(c.Request.Context(),`SELECT COUNT(DISTINCT LOWER(TRIM(email))) `+productEmailAudience).Scan(&eligible);err!=nil {
        log.Printf("count product email audience: %v",err)
        c.JSON(http.StatusInternalServerError,gin.H{"error":"Could not load audience"});return
    }
    c.JSON(http.StatusOK,gin.H{"eligible":eligible})
}

func ExportProductEmailAudience(c *gin.Context) {
    c.Header("Cache-Control", "no-store")
    db := c.MustGet("db").(*sql.DB)
    rows,err:=db.QueryContext(c.Request.Context(),`SELECT DISTINCT LOWER(TRIM(email)) `+productEmailAudience+` ORDER BY 1`)
    if err!=nil {log.Printf("export product emails: %v",err);c.JSON(500,gin.H{"error":"Could not export audience"});return}
    defer rows.Close()
    emails:=make([]string,0)
    for rows.Next() {
        var address string
        if err:=rows.Scan(&address);err!=nil {log.Printf("scan product emails: %v",err);c.JSON(500,gin.H{"error":"Could not export audience"});return}
        // Reject malformed and spreadsheet-formula values before CSV export.
        if !strings.Contains(address,"@") || strings.ContainsAny(address,"\r\n\t") || strings.ContainsAny(address[:1],"=+-@") {continue}
        emails=append(emails,address)
    }
    if err:=rows.Err();err!=nil {log.Printf("iterate product emails: %v",err);c.JSON(500,gin.H{"error":"Could not export audience"});return}
    c.Header("Content-Disposition",`attachment; filename="l2e-product-email-audience.csv"`)
    c.Header("Content-Type","text/csv; charset=utf-8")
    writer:=csv.NewWriter(c.Writer)
    _=writer.Write([]string{"EMAIL"})
    for _,address:=range emails {_=writer.Write([]string{address})}
    writer.Flush()
}
