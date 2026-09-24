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

func AuthRequired(c *gin.Context) {
    authHeader := c.GetHeader("Authorization")
    tokenString:=""
    cookieAuth:=false
    parts:=strings.Fields(authHeader)
    if len(parts)==2 && parts[0]=="Bearer" && !strings.HasPrefix(parts[1],"cookie-session") {tokenString=parts[1]}
    if tokenString=="" {
        var err error
        tokenString,err=c.Cookie(SessionCookieName)
        if err!=nil||tokenString=="" {c.JSON(http.StatusUnauthorized,gin.H{"error":"Sign in required"});c.Abort();return}
        cookieAuth=true
    }
    if cookieAuth && c.Request.Method!="GET" && c.Request.Method!="HEAD" && c.Request.Method!="OPTIONS" && !SameOrigin(c) {
        c.JSON(http.StatusForbidden,gin.H{"error":"Invalid request origin"});c.Abort();return
    }

    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if token.Method.Alg()!=jwt.SigningMethodHS256.Alg(){return nil,http.ErrNoCookie}
        return config.JWTSecret(), nil
    })

    if err != nil || !token.Valid {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
        c.Abort()
        return
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
        c.Abort()
        return
    }

    value,valid:=claims["user_id"].(float64)
    if !valid||value<=0 {c.JSON(http.StatusUnauthorized,gin.H{"error":"Invalid token claims"});c.Abort();return}
    userID := int64(value)
    c.Set("user_id", userID)
    c.Next()
}
