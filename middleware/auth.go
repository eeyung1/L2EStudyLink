package middleware

import (
    "net/http"
    "net/url"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"

    "L2EStudyLink/config"
)

const SessionCookieName = "l2e_session"

func SameOrigin(c *gin.Context) bool {
    origin:=c.GetHeader("Origin")
    parsed,err:=url.Parse(origin)
    return err==nil && (parsed.Scheme=="https"||parsed.Scheme=="http") && strings.EqualFold(parsed.Host,c.Request.Host)
}

// The API and HTML guards share token and origin validation.
func AuthRequired(c *gin.Context) {
    userID,status,message:=authenticatedUser(c)
    if status!=0 {c.AbortWithStatusJSON(status,gin.H{"error":message});return}
    c.Set("user_id",userID);c.Next()
}

// HTML navigation recovers to the account form; API failures remain JSON.
func PageAuthRequired(c *gin.Context) {
    c.Header("Cache-Control","private, no-store")
    userID,status,_:=authenticatedUser(c)
    if status!=0 {c.Redirect(http.StatusSeeOther,"/page/login");c.Abort();return}
    c.Set("user_id",userID);c.Next()
}

func authenticatedUser(c *gin.Context) (int64,int,string) {
    tokenString:="";cookieAuth:=false;parts:=strings.Fields(c.GetHeader("Authorization"))
    if len(parts)==2 && parts[0]=="Bearer" && !strings.HasPrefix(parts[1],"cookie-session") {tokenString=parts[1]}
    if tokenString=="" {
        var err error;tokenString,err=c.Cookie(SessionCookieName)
        if err!=nil||tokenString=="" {return 0,http.StatusUnauthorized,"Sign in required"}
        cookieAuth=true
    }
    if cookieAuth && c.Request.Method!="GET" && c.Request.Method!="HEAD" && c.Request.Method!="OPTIONS" && !SameOrigin(c) {return 0,http.StatusForbidden,"Invalid request origin"}
    token,err:=jwt.Parse(tokenString,func(token *jwt.Token)(interface{},error){
        if token.Method.Alg()!=jwt.SigningMethodHS256.Alg(){return nil,http.ErrNoCookie}
        return config.JWTSecret(),nil
    })
    if err!=nil||!token.Valid {return 0,http.StatusUnauthorized,"Invalid or expired token"}
    claims,ok:=token.Claims.(jwt.MapClaims);if !ok {return 0,http.StatusUnauthorized,"Invalid token claims"}
    value,valid:=claims["user_id"].(float64);if !valid||value<=0 {return 0,http.StatusUnauthorized,"Invalid token claims"}
    return int64(value),0,""
}
