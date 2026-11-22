package handlers

import (
    "database/sql"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"

    "L2EStudyLink/middleware"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type VoiceResponse struct {
    AudioURL string `json:"audio_url"`
    Text string `json:"text"`
}

var voiceFilename = regexp.MustCompile(`^(?:[1-9][0-9]*-)?[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\.(webm|ogg|mp4|wav|m4a|mp3)$`)

func voiceDirectory() string {
    if directory := os.Getenv("VOICE_UPLOAD_DIR"); directory != "" { return directory }
    return "uploads"
}

// Both current and legacy URLs stay stable, but the public static mount is gone.
func RegisterVoiceRoutes(router *gin.Engine) {
    router.POST("/api/v1/voice/upload", middleware.AuthRequired, HandleVoiceUpload)
    router.POST("/api/voice/upload", middleware.AuthRequired, HandleVoiceUpload)
    router.GET("/uploads/:filename", middleware.AuthRequired, ServeVoiceUpload)
    router.HEAD("/uploads/:filename", middleware.AuthRequired, ServeVoiceUpload)
}

func HandleVoiceUpload(c *gin.Context) {
    userID := c.GetInt64("user_id")
    if userID <= 0 { c.JSON(http.StatusUnauthorized, gin.H{"error":"Sign in required"}); return }
    c.Header("Cache-Control", "private, no-store")
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)
    file, err := c.FormFile("audio")
    if err != nil || file.Size == 0 { c.JSON(400,gin.H{"error":"Choose a non-empty recording under 5 MB"}); return }
    extension := strings.ToLower(filepath.Ext(file.Filename))
    switch extension { case ".webm", ".ogg", ".mp4", ".wav", ".m4a", ".mp3": default: c.JSON(400,gin.H{"error":"Unsupported recording format"});return }
    if err = os.MkdirAll(voiceDirectory(),0700);err != nil {c.JSON(500,gin.H{"error":"Unable to store recording"});return}
    filename := fmt.Sprintf("%d-%s%s",userID,uuid.New().String(),extension)
    path := filepath.Join(voiceDirectory(),filename)
    source,err:=file.Open();if err!=nil {c.JSON(500,gin.H{"error":"Unable to read recording"});return};defer source.Close()
    target,err:=os.OpenFile(path,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600)
    if err!=nil {c.JSON(500,gin.H{"error":"Unable to store recording"});return}
    _,copyErr:=io.Copy(target,source);closeErr:=target.Close()
    if copyErr!=nil||closeErr!=nil {os.Remove(path);c.JSON(500,gin.H{"error":"Unable to store recording"});return}
    c.JSON(200,VoiceResponse{AudioURL:"/uploads/"+filename,Text:""})
}

func voiceOwned(c *gin.Context, raw string) bool {
    userID:=c.GetInt64("user_id")
    if userID<=0||!strings.HasPrefix(raw,"/uploads/"){return false}
    name:=strings.TrimPrefix(raw,"/uploads/")
    if !voiceFilename.MatchString(name){return false}
    // A numeric prefix belongs to the authenticated uploader; never trust a submitted owner ID.
    if len(strings.TrimSuffix(name,filepath.Ext(name)))>36 {
        prefix:=strings.SplitN(name,"-",2)[0];return prefix==strconv.FormatInt(userID,10)
    }
    database,ok:=c.Get("db");if !ok{return false}
    var allowed bool
    err:=database.(*sql.DB).QueryRowContext(c.Request.Context(),`SELECT EXISTS(SELECT 1 FROM activity_logs WHERE user_id=$1 AND audio_url=$2)`,userID,raw).Scan(&allowed)
    return err==nil&&allowed
}

func voiceExists(raw string) bool {
    name:=strings.TrimPrefix(raw,"/uploads/")
    if !voiceFilename.MatchString(name){return false}
    info,err:=os.Lstat(filepath.Join(voiceDirectory(),name))
    return err==nil && info.Mode().IsRegular()
}

func ServeVoiceUpload(c *gin.Context) {
    c.Header("Cache-Control","private, no-store")
    c.Header("X-Content-Type-Options","nosniff")
    c.Header("Content-Security-Policy","default-src 'none'; sandbox")
    raw:="/uploads/"+c.Param("filename")
    if !voiceOwned(c,raw)||!voiceExists(raw){c.Status(404);return}
    types:=map[string]string{".webm":"audio/webm",".ogg":"audio/ogg",".mp4":"audio/mp4",".wav":"audio/wav",".m4a":"audio/mp4",".mp3":"audio/mpeg"}
    c.Header("Content-Type",types[filepath.Ext(raw)])
    c.File(filepath.Join(voiceDirectory(),c.Param("filename")))
}
