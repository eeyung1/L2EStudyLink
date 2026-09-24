package middleware

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"

 "github.com/gin-gonic/gin"
 "github.com/golang-jwt/jwt/v5"
)

func TestCookieAuthAndSameOriginWrites(t *testing.T) {
 t.Setenv("JWT_SECRET","test-cookie-secret-for-ci")
 token,err:=jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.MapClaims{"user_id":float64(42),"exp":time.Now().Add(time.Hour).Unix()}).SignedString([]byte("test-cookie-secret-for-ci"))
 if err!=nil {t.Fatal(err)}
 r:=gin.New();r.GET("/me",AuthRequired,func(c *gin.Context){c.JSON(200,gin.H{"id":c.GetInt64("user_id")})});r.POST("/save",AuthRequired,func(c *gin.Context){c.Status(204)})
 call:=func(method,path,origin string,cookie,bearer bool) int {t.Helper();w:=httptest.NewRecorder();req:=httptest.NewRequest(method,"https://example.test"+path,strings.NewReader(""));if origin!="" {req.Header.Set("Origin",origin)};if cookie {req.AddCookie(&http.Cookie{Name:SessionCookieName,Value:token})};if bearer {req.Header.Set("Authorization","Bearer "+token)} else if cookie {req.Header.Set("Authorization","Bearer cookie-session-scope")};r.ServeHTTP(w,req);return w.Code}
 if got:=call("GET","/me","",true,false);got!=200 {t.Fatalf("cookie read: %d",got)}
 if got:=call("POST","/save","",true,false);got!=403 {t.Fatalf("missing origin: %d",got)}
 if got:=call("POST","/save","https://evil.test",true,false);got!=403 {t.Fatalf("foreign origin: %d",got)}
 if got:=call("POST","/save","https://example.test",true,false);got!=204 {t.Fatalf("same-origin write: %d",got)}
 if got:=call("GET","/me","",false,false);got!=401 {t.Fatalf("missing cookie: %d",got)}
 if got:=call("POST","/save","",false,true);got!=204 {t.Fatalf("legacy bearer: %d",got)}
}
