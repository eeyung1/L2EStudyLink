package handlers

import (
 "crypto/rand"
 "encoding/hex"
 "net/http"
 "strings"

 "github.com/gin-gonic/gin"
 "L2EStudyLink/middleware"
)

func cookieSecure(c *gin.Context) bool {
 return c.Request.TLS!=nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"),"https")
}

func SetSession(c *gin.Context, token string) error {
 var scope [16]byte
 if _,err:=rand.Read(scope[:]);err!=nil {return err}
 c.SetSameSite(http.SameSiteStrictMode)
 c.SetCookie(middleware.SessionCookieName,token,24*60*60,"/","",cookieSecure(c),true)
 c.SetCookie("l2e_scope",hex.EncodeToString(scope[:]),24*60*60,"/","",cookieSecure(c),false)
 return nil
}

func Logout(c *gin.Context) {
 if !middleware.SameOrigin(c) {c.JSON(http.StatusForbidden,gin.H{"error":"Invalid request origin"});return}
 c.SetSameSite(http.SameSiteStrictMode)
 c.SetCookie(middleware.SessionCookieName,"",-1,"/","",cookieSecure(c),true)
 c.SetCookie("l2e_scope","",-1,"/","",cookieSecure(c),false)
 c.JSON(http.StatusOK,gin.H{"message":"Signed out"})
}

// Existing bearer sessions are exchanged for cookies on the next signed-in
// page visit. Bearer tokens expire naturally after their original 24 hours.
func MigrateSession(c *gin.Context) {
 parts:=strings.Fields(c.GetHeader("Authorization"))
 if len(parts)!=2||parts[0]!="Bearer"||strings.HasPrefix(parts[1],"cookie-session") {c.JSON(400,gin.H{"error":"Legacy token required"});return}
 if !middleware.SameOrigin(c) {c.JSON(403,gin.H{"error":"Invalid request origin"});return}
 if err:=SetSession(c,parts[1]);err!=nil {c.JSON(500,gin.H{"error":"Could not move session"});return}
 c.JSON(200,gin.H{"message":"Session moved"})
}
