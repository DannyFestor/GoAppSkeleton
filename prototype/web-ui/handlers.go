package main

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"strconv"
)

//go:embed templates/*.html
var templateFiles embed.FS

var pages = template.Must(template.New("").Funcs(template.FuncMap{
	"add":            func(a, b int) int { return a + b },
	"until":          func(n int) []int { return make([]int, n) },
	"AppliesBecause": AppliesBecause,
}).ParseFS(templateFiles, "templates/*.html"))

func handlePage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	step, _ := strconv.Atoi(query.Get("step"))
	page := newPage(Resolve(stateFromQuery(query)), query.Get("variant"), step)
	if err := pages.ExecuteTemplate(w, "layout", page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleConfigurationFile(w http.ResponseWriter, r *http.Request) {
	resolved := Resolve(stateFromQuery(r.URL.Query()))
	w.Header().Set("Content-Type", "application/toml")
	w.Header().Set("Content-Disposition", `attachment; filename="`+resolved.ConfigurationFileName()+`"`)
	_, _ = io.WriteString(w, resolved.ConfigurationTOML())
}

func handleZip(w http.ResponseWriter, r *http.Request) {
	resolved := Resolve(stateFromQuery(r.URL.Query()))
	if len(resolved.Errors) > 0 {
		http.Error(w, "Configuration has errors; fix them in the form first", http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+resolved.ProjectName()+`.zip"`)
	_ = resolved.WriteZip(w)
}

func handleLoad(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("configuration")
	if err != nil {
		http.Error(w, "choose a Configuration file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	content, _ := io.ReadAll(file)
	query := stateFromConfigurationFile(string(content)).Query()
	query.Set("variant", r.FormValue("variant"))
	http.Redirect(w, r, "/?"+query.Encode(), http.StatusSeeOther)
}
