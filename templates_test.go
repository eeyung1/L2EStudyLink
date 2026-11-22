package main

import (
    "bytes"
    "html/template"
    "strings"
    "testing"
)

func TestWorkspaceNavigationTemplate(t *testing.T) {
    pages, err := template.ParseGlob("templates/*.html")
    if err != nil { t.Fatal(err) }
    for _, page := range []string{"dashboard.html", "timetable.html", "planner.html", "focus.html", "buddy.html", "skills.html", "availability.html", "my-bookings.html", "search-page.html", "projects.html", "masterclass.html", "reflections.html", "admin.html"} {
        t.Run(page, func(t *testing.T) {
            var output bytes.Buffer
            if err := pages.ExecuteTemplate(&output,page,nil); err != nil { t.Fatal(err) }
            html := output.String()
            if strings.Count(html,`id="mobile-navigation"`) != 1 { t.Fatal("expected exactly one shared navigation") }
            for _, path := range []string{"/planner", "/focus", "/buddies"} {
                if !strings.Contains(html,`href="`+path+`"`) { t.Errorf("missing navigation to %s",path) }
            }
        })
    }
}
