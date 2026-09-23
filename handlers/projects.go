package handlers

import (
    "database/sql"
    "errors"
    "net/http"
    "net/url"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
)

func projectDB(c *gin.Context) *sql.DB { return c.MustGet("db").(*sql.DB) }
func projectID(c *gin.Context) (int64, bool) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || id < 1 { c.JSON(http.StatusBadRequest, gin.H{"error":"Invalid ID"}); return 0, false }
    return id, true
}
func projectActive(c *gin.Context) bool {
    var active bool
    err := projectDB(c).QueryRowContext(c.Request.Context(), `SELECT NOT is_suspended FROM users WHERE id=$1`, c.GetInt64("user_id")).Scan(&active)
    if err != nil || !active { c.JSON(http.StatusForbidden, gin.H{"error":"Account unavailable"}); return false }
    return true
}

func GetCollaborationPreference(c *gin.Context) {
    var open bool
    err := projectDB(c).QueryRowContext(c.Request.Context(), `SELECT open_to_invites FROM collaboration_preferences WHERE user_id=$1`, c.GetInt64("user_id")).Scan(&open)
    if err != nil && !errors.Is(err, sql.ErrNoRows) { c.JSON(500, gin.H{"error":"Could not load preference"}); return }
    c.JSON(200, gin.H{"open_to_invites":open})
}
func SetCollaborationPreference(c *gin.Context) {
    if !projectActive(c) { return }
    var input struct { Open bool `json:"open_to_invites"` }
    if err := c.ShouldBindJSON(&input); err != nil { c.JSON(400, gin.H{"error":"Invalid preference"}); return }
    _, err := projectDB(c).ExecContext(c.Request.Context(), `INSERT INTO collaboration_preferences(user_id,open_to_invites) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET open_to_invites=EXCLUDED.open_to_invites`, c.GetInt64("user_id"), input.Open)
    if err != nil { c.JSON(500, gin.H{"error":"Could not save preference"}); return }
    c.JSON(200, gin.H{"open_to_invites":input.Open})
}
func ListCollaborators(c *gin.Context) {
    rows, err := projectDB(c).QueryContext(c.Request.Context(), `SELECT u.id,u.name,COALESCE(u.bio,''),COALESCE(string_agg(DISTINCT s.skill_name, ', '),'') FROM users u JOIN collaboration_preferences cp ON cp.user_id=u.id LEFT JOIN skills s ON s.user_id=u.id WHERE cp.open_to_invites=TRUE AND u.is_suspended=FALSE AND u.id<>$1 GROUP BY u.id,u.name,u.bio ORDER BY u.name LIMIT 100`, c.GetInt64("user_id"))
    if err != nil { c.JSON(500, gin.H{"error":"Could not load collaborators"}); return }
    defer rows.Close()
    result := []gin.H{}
    for rows.Next() { var id int64; var name,bio,skills string; if err:=rows.Scan(&id,&name,&bio,&skills); err!=nil { c.JSON(500,gin.H{"error":"Could not load collaborators"}); return }; result=append(result,gin.H{"id":id,"name":name,"bio":bio,"skills":skills}) }
    if rows.Err()!=nil { c.JSON(500,gin.H{"error":"Could not load collaborators"}); return }
    c.JSON(200,result)
}

func CreateProject(c *gin.Context) {
    if !projectActive(c) { return }
    var input struct { Title string `json:"title"`; Description string `json:"description"`; Roles string `json:"roles_needed"`; Commitment string `json:"time_commitment"`; RepoURL string `json:"repo_url"` }
    if c.ShouldBindJSON(&input)!=nil { c.JSON(400,gin.H{"error":"Invalid project"}); return }
    input.Title=strings.TrimSpace(input.Title); input.Description=strings.TrimSpace(input.Description); input.Roles=strings.TrimSpace(input.Roles); input.Commitment=strings.TrimSpace(input.Commitment); input.RepoURL=strings.TrimSpace(input.RepoURL)
    if len(input.Title)<3 || len(input.Title)>120 || len(input.Description)<20 || len(input.Description)>3000 || len(input.Roles)<2 || len(input.Roles)>240 || len(input.Commitment)<2 || len(input.Commitment)>120 || len(input.RepoURL)>500 { c.JSON(400,gin.H{"error":"Complete the project details within the field limits"}); return }
    if input.RepoURL!="" { u,e:=url.Parse(input.RepoURL); if e!=nil || (u.Scheme!="https" && u.Scheme!="http") || u.Host=="" { c.JSON(400,gin.H{"error":"Use a valid http or https repository link"}); return } }
    var id int64
    err:=projectDB(c).QueryRowContext(c.Request.Context(),`INSERT INTO projects(owner_id,title,description,roles_needed,time_commitment,repo_url) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,c.GetInt64("user_id"),input.Title,input.Description,input.Roles,input.Commitment,input.RepoURL).Scan(&id)
    if err!=nil { c.JSON(500,gin.H{"error":"Could not create project"}); return }
    c.JSON(201,gin.H{"id":id})
}
func ListProjects(c *gin.Context) {
    rows,err:=projectDB(c).QueryContext(c.Request.Context(),`SELECT p.id,p.owner_id,u.name,p.title,p.description,p.roles_needed,p.time_commitment,p.repo_url,p.status,p.created_at,(SELECT count(*) FROM project_members m WHERE m.project_id=p.id) FROM projects p JOIN users u ON u.id=p.owner_id WHERE p.status='open' OR p.owner_id=$1 OR EXISTS(SELECT 1 FROM project_members m WHERE m.project_id=p.id AND m.user_id=$1) ORDER BY p.created_at DESC LIMIT 100`,c.GetInt64("user_id"))
    if err!=nil { c.JSON(500,gin.H{"error":"Could not load projects"}); return }; defer rows.Close()
    items:=[]gin.H{}
    for rows.Next(){var id,owner int64;var name,title,description,roles,commitment,repo,status string;var date interface{};var members int; if rows.Scan(&id,&owner,&name,&title,&description,&roles,&commitment,&repo,&status,&date,&members)!=nil {c.JSON(500,gin.H{"error":"Could not load projects"});return};items=append(items,gin.H{"id":id,"owner_id":owner,"owner_name":name,"title":title,"description":description,"roles_needed":roles,"time_commitment":commitment,"repo_url":repo,"status":status,"created_at":date,"member_count":members})}
    if rows.Err()!=nil {c.JSON(500,gin.H{"error":"Could not load projects"});return};c.JSON(200,items)
}
func GetProject(c *gin.Context) {
    id,ok:=projectID(c);if !ok{return};db:=projectDB(c);var owner int64;var name,title,description,roles,commitment,repo,status string
    err:=db.QueryRowContext(c.Request.Context(),`SELECT p.owner_id,u.name,p.title,p.description,p.roles_needed,p.time_commitment,p.repo_url,p.status FROM projects p JOIN users u ON u.id=p.owner_id WHERE p.id=$1 AND (p.status='open' OR p.owner_id=$2 OR EXISTS(SELECT 1 FROM project_members m WHERE m.project_id=p.id AND m.user_id=$2))`,id,c.GetInt64("user_id")).Scan(&owner,&name,&title,&description,&roles,&commitment,&repo,&status)
    if errors.Is(err,sql.ErrNoRows){c.JSON(404,gin.H{"error":"Project not found"});return};if err!=nil {c.JSON(500,gin.H{"error":"Could not load project"});return}
    members:=[]gin.H{};rows,err:=db.QueryContext(c.Request.Context(),`SELECT u.id,u.name FROM project_members m JOIN users u ON u.id=m.user_id WHERE m.project_id=$1 ORDER BY u.name`,id)
    if err!=nil {c.JSON(500,gin.H{"error":"Could not load members"});return};for rows.Next(){var uid int64;var n string;if rows.Scan(&uid,&n)!=nil{rows.Close();c.JSON(500,gin.H{"error":"Could not load members"});return};members=append(members,gin.H{"id":uid,"name":n})};err=rows.Err();rows.Close();if err!=nil {c.JSON(500,gin.H{"error":"Could not load members"});return}
    var requestID int64;var requestStatus string;var requestKind string
    err=db.QueryRowContext(c.Request.Context(),`SELECT id,status,kind FROM project_requests WHERE project_id=$1 AND user_id=$2`,id,c.GetInt64("user_id")).Scan(&requestID,&requestStatus,&requestKind)
    if err!=nil && !errors.Is(err,sql.ErrNoRows){c.JSON(500,gin.H{"error":"Could not load request"});return}
    c.JSON(200,gin.H{"id":id,"owner_id":owner,"owner_name":name,"title":title,"description":description,"roles_needed":roles,"time_commitment":commitment,"repo_url":repo,"status":status,"members":members,"my_request_id":requestID,"my_request_status":requestStatus,"my_request_kind":requestKind})
}
func SetProjectStatus(c *gin.Context) {
    if !projectActive(c){return};id,ok:=projectID(c);if !ok{return};var input struct{Status string `json:"status"`};if c.ShouldBindJSON(&input)!=nil || (input.Status!="open" && input.Status!="closed"){c.JSON(400,gin.H{"error":"Invalid status"});return}
    result,err:=projectDB(c).ExecContext(c.Request.Context(),`UPDATE projects SET status=$1 WHERE id=$2 AND owner_id=$3`,input.Status,id,c.GetInt64("user_id"));if err!=nil{c.JSON(500,gin.H{"error":"Could not update project"});return};n,_:=result.RowsAffected();if n==0{c.JSON(404,gin.H{"error":"Project not found or not owned"});return};c.JSON(200,gin.H{"status":input.Status})
}

func CreateProjectRequest(c *gin.Context) {
    if !projectActive(c){return};id,ok:=projectID(c);if !ok{return}
    var input struct { Kind string `json:"kind"`; UserID int64 `json:"user_id"`; Note string `json:"note"` }
    if c.ShouldBindJSON(&input)!=nil{c.JSON(400,gin.H{"error":"Invalid request"});return}
    input.Note=strings.TrimSpace(input.Note)
    if len(input.Note)>500{c.JSON(400,gin.H{"error":"Note is too long"});return}
    db:=projectDB(c);actor:=c.GetInt64("user_id");var owner int64;var status string
    err:=db.QueryRowContext(c.Request.Context(),`SELECT owner_id,status FROM projects WHERE id=$1`,id).Scan(&owner,&status)
    if errors.Is(err,sql.ErrNoRows){c.JSON(404,gin.H{"error":"Project not found"});return};if err!=nil{c.JSON(500,gin.H{"error":"Could not load project"});return}
    if status!="open"{c.JSON(409,gin.H{"error":"Project is closed"});return}
    var target int64
    switch input.Kind {
    case "application":
        if actor==owner || len(input.Note)<10{c.JSON(400,gin.H{"error":"Explain how you can contribute"});return};target=actor
    case "invitation":
        if actor!=owner{c.JSON(403,gin.H{"error":"Only the owner can invite"});return};target=input.UserID
        if target==owner || target<1{c.JSON(400,gin.H{"error":"Choose another fellow"});return}
        var allowed bool
        err=db.QueryRowContext(c.Request.Context(),`SELECT EXISTS(SELECT 1 FROM collaboration_preferences cp JOIN users u ON u.id=cp.user_id WHERE cp.user_id=$1 AND cp.open_to_invites=TRUE AND u.is_suspended=FALSE)`,target).Scan(&allowed)
        if err!=nil{c.JSON(500,gin.H{"error":"Could not check invitation preference"});return};if !allowed{c.JSON(403,gin.H{"error":"This fellow is not open to invitations"});return}
    default:c.JSON(400,gin.H{"error":"Invalid request type"});return
    }
    var member bool
    err=db.QueryRowContext(c.Request.Context(),`SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id=$1 AND user_id=$2)`,id,target).Scan(&member)
    if err!=nil{c.JSON(500,gin.H{"error":"Could not check membership"});return};if member{c.JSON(409,gin.H{"error":"Already on this project"});return}
    var requestID int64
    err=db.QueryRowContext(c.Request.Context(),`INSERT INTO project_requests(project_id,user_id,kind,note) VALUES($1,$2,$3,$4) ON CONFLICT(project_id,user_id) DO NOTHING RETURNING id`,id,target,input.Kind,input.Note).Scan(&requestID)
    if errors.Is(err,sql.ErrNoRows){c.JSON(409,gin.H{"error":"A request already exists for this fellow"});return};if err!=nil{c.JSON(500,gin.H{"error":"Could not send request"});return}
    c.JSON(201,gin.H{"id":requestID,"status":"pending"})
}
func ListProjectRequests(c *gin.Context) {
    rows,err:=projectDB(c).QueryContext(c.Request.Context(),`SELECT r.id,r.project_id,p.title,r.user_id,u.name,r.kind,r.note,r.status,p.owner_id FROM project_requests r JOIN projects p ON p.id=r.project_id JOIN users u ON u.id=r.user_id WHERE (r.kind='invitation' AND (r.user_id=$1 OR p.owner_id=$1)) OR (r.kind='application' AND p.owner_id=$1) ORDER BY r.created_at DESC LIMIT 100`,c.GetInt64("user_id"))
    if err!=nil{c.JSON(500,gin.H{"error":"Could not load requests"});return};defer rows.Close();items:=[]gin.H{}
    for rows.Next(){var id,pid,uid,owner int64;var title,name,kind,note,status string;if rows.Scan(&id,&pid,&title,&uid,&name,&kind,&note,&status,&owner)!=nil{c.JSON(500,gin.H{"error":"Could not load requests"});return};items=append(items,gin.H{"id":id,"project_id":pid,"project_title":title,"user_id":uid,"user_name":name,"kind":kind,"note":note,"status":status,"owner_id":owner})}
    if rows.Err()!=nil{c.JSON(500,gin.H{"error":"Could not load requests"});return};c.JSON(200,items)
}
func RespondProjectRequest(c *gin.Context) {
    if !projectActive(c){return};id,ok:=projectID(c);if !ok{return}
    var input struct{Status string `json:"status"`};if c.ShouldBindJSON(&input)!=nil || (input.Status!="accepted" && input.Status!="declined"){c.JSON(400,gin.H{"error":"Choose accepted or declined"});return}
    tx,err:=projectDB(c).BeginTx(c.Request.Context(),nil);if err!=nil{c.JSON(500,gin.H{"error":"Could not update request"});return};defer tx.Rollback()
    var pid,uid,owner int64;var kind,status,projectStatus string
    err=tx.QueryRowContext(c.Request.Context(),`SELECT r.project_id,r.user_id,p.owner_id,r.kind,r.status,p.status FROM project_requests r JOIN projects p ON p.id=r.project_id WHERE r.id=$1 FOR UPDATE OF r,p`,id).Scan(&pid,&uid,&owner,&kind,&status,&projectStatus)
    if errors.Is(err,sql.ErrNoRows){c.JSON(404,gin.H{"error":"Request not found"});return};if err!=nil{c.JSON(500,gin.H{"error":"Could not load request"});return}
    actor:=c.GetInt64("user_id")
    if (kind=="application" && actor!=owner) || (kind=="invitation" && actor!=uid){c.JSON(403,gin.H{"error":"Not your request to decide"});return}
    if status!="pending"{c.JSON(409,gin.H{"error":"Request already decided"});return}
    if input.Status=="accepted" && projectStatus!="open"{c.JSON(409,gin.H{"error":"Project is closed"});return}
    if input.Status=="accepted" {
        if _,err=tx.ExecContext(c.Request.Context(),`INSERT INTO project_members(project_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`,pid,uid);err!=nil{c.JSON(500,gin.H{"error":"Could not join project"});return}
    }
    if _,err=tx.ExecContext(c.Request.Context(),`UPDATE project_requests SET status=$1 WHERE id=$2`,input.Status,id);err!=nil{c.JSON(500,gin.H{"error":"Could not update request"});return}
    if err=tx.Commit();err!=nil{c.JSON(500,gin.H{"error":"Could not save decision"});return};c.JSON(200,gin.H{"status":input.Status})
}
