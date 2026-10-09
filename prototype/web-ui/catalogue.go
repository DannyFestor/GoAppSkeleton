package main

// PROTOTYPE: the core tier of the catalogue from the "Tiered Option catalogue"
// ticket, hard-coded in Go. The real Generator reads option.toml files.

type Kind int

const (
	Single Kind = iota
	Multi
	Toggle
)

// Cond matches when Option's selection contains any of Choices, or, with no
// Choices, when the Option is anything other than none or the empty set.
type Cond struct {
	Option  string
	Choices []string
}

type Choice struct {
	ID       string
	Label    string
	Summary  string
	Requires []Cond
	Files    []string
}

type ConditionalDefault struct {
	When   Cond
	Choice string
}

type Option struct {
	ID                  string
	Label               string
	Summary             string
	Kind                Kind
	Choices             []Choice
	Default             []string
	ConditionalDefaults []ConditionalDefault
	AppliesWhen         []Cond
}

type Group struct {
	Label   string
	Options []*Option
}

var webOnly = []Cond{{Option: "app-type", Choices: []string{"web"}}}
var withDatabase = []Cond{{Option: "database"}}

func toggleChoices(summary string, files ...string) []Choice {
	return []Choice{
		{ID: "none", Label: "Off", Summary: "Leave it out"},
		{ID: "on", Label: "On", Summary: summary, Files: files},
	}
}

var catalogue = []Group{
	{Label: "Application", Options: []*Option{
		{ID: "app-type", Label: "App type", Summary: "What the Skeleton serves", Kind: Single, Default: []string{"web"}, Choices: []Choice{
			{ID: "web", Label: "Web", Summary: "Server-rendered HTML pages", Files: []string{"ui/ui.go", "ui/templates/layouts/base.html", "ui/templates/pages/home.html", "public/css/main.css"}},
			{ID: "api", Label: "API", Summary: "JSON over HTTP"},
		}},
		{ID: "router", Label: "Router", Summary: "How requests reach handlers", Kind: Single, Default: []string{"servemux"}, Choices: []Choice{
			{ID: "servemux", Label: "http.ServeMux", Summary: "Standard library router"},
			{ID: "chi", Label: "chi", Summary: "Lightweight, net/http-compatible router"},
		}},
		{ID: "templating", Label: "Templating", Summary: "How HTML is rendered", Kind: Single, Default: []string{"html-template"}, AppliesWhen: webOnly, Choices: []Choice{
			{ID: "html-template", Label: "html/template", Summary: "Standard library templates"},
			{ID: "templ", Label: "templ", Summary: "Type-checked components compiled to Go", Files: []string{"ui/components/layout.templ"}},
		}},
		{ID: "htmx", Label: "HTMX", Summary: "Partial page updates from HTML attributes", Kind: Toggle, Default: []string{"none"}, AppliesWhen: webOnly, Choices: toggleChoices("Vendored htmx.min.js and a partial-render helper", "public/js/htmx.min.js")},
		{ID: "custom-error-pages", Label: "Custom error pages", Summary: "Styled 404 and 500 pages", Kind: Toggle, Default: []string{"none"}, AppliesWhen: webOnly, Choices: toggleChoices("Templates for 404 and 500", "ui/templates/pages/404.html", "ui/templates/pages/500.html")},
	}},
	{Label: "Configuration and logging", Options: []*Option{
		{ID: "config-source", Label: "Configuration sources", Summary: "Where settings come from besides env", Kind: Multi, Default: []string{}, Choices: []Choice{
			{ID: "flags", Label: "Flags", Summary: "Command-line flags"},
			{ID: "file", Label: "File", Summary: "A config file next to the binary"},
			{ID: "dotenv", Label: ".env", Summary: "Load .env into unset variables", Files: []string{".env.example"}},
		}},
		{ID: "config-file-format", Label: "Config file format", Summary: "Format of the app's config file", Kind: Single, Default: []string{"toml"}, AppliesWhen: []Cond{{Option: "config-source", Choices: []string{"file"}}}, Choices: []Choice{
			{ID: "toml", Label: "TOML", Summary: "BurntSushi/toml", Files: []string{"config.example.toml"}},
			{ID: "yaml", Label: "YAML", Summary: "go.yaml.in/yaml/v4", Files: []string{"config.example.yaml"}},
			{ID: "json", Label: "JSON", Summary: "encoding/json", Files: []string{"config.example.json"}},
		}},
		{ID: "cli", Label: "CLI", Summary: "How the binary parses commands", Kind: Single, Default: []string{"stdlib"}, Choices: []Choice{
			{ID: "stdlib", Label: "Standard library", Summary: "One command, flag package"},
			{ID: "cobra", Label: "cobra", Summary: "serve, version and migrate subcommands", Files: []string{"cmd/web/root.go", "cmd/web/serve.go", "cmd/web/version.go"}},
		}},
		{ID: "log-format", Label: "Log format", Summary: "How slog writes records", Kind: Single, Default: []string{"text"}, Choices: []Choice{
			{ID: "text", Label: "Text", Summary: "slog.TextHandler"},
			{ID: "json", Label: "JSON", Summary: "slog.JSONHandler"},
			{ID: "color", Label: "Colour", Summary: "Coloured text for terminals"},
		}},
		{ID: "request-logging", Label: "Request logging", Summary: "One log line per request", Kind: Toggle, Default: []string{"on"}, Choices: toggleChoices("Logging middleware", "internal/httpx/request_logger.go")},
	}},
	{Label: "Data", Options: []*Option{
		{ID: "database", Label: "Database", Summary: "The relational store", Kind: Single, Default: []string{"none"}, Choices: []Choice{
			{ID: "none", Label: "None", Summary: "No database"},
			{ID: "sqlite", Label: "SQLite", Summary: "modernc.org/sqlite, no cgo", Files: []string{"internal/database/database.go"}},
			{ID: "postgres", Label: "PostgreSQL", Summary: "pgx", Files: []string{"internal/database/database.go"}},
			{ID: "mysql", Label: "MySQL", Summary: "go-sql-driver/mysql", Files: []string{"internal/database/database.go"}},
		}},
		{ID: "data-access", Label: "Data access", Summary: "How queries are written", Kind: Single, Default: []string{"database-sql"}, AppliesWhen: withDatabase, Choices: []Choice{
			{ID: "database-sql", Label: "database/sql", Summary: "Standard library"},
			{ID: "sqlx", Label: "sqlx", Summary: "Struct scanning on top of database/sql"},
			{ID: "sqlc", Label: "sqlc", Summary: "Go generated from SQL", Files: []string{"sqlc.yaml", "queries/.gitkeep"}},
		}},
		{ID: "migrations", Label: "Migrations", Summary: "Schema changes over time", Kind: Single, Default: []string{"goose"}, AppliesWhen: withDatabase, Choices: []Choice{
			{ID: "none", Label: "None", Summary: "Manage the schema yourself"},
			{ID: "goose", Label: "goose", Summary: "SQL migrations, embedded", Files: []string{"migrations/0001_init.sql"}},
			{ID: "golang-migrate", Label: "golang-migrate", Summary: "Up and down SQL files", Files: []string{"migrations/0001_init.up.sql", "migrations/0001_init.down.sql"}},
		}},
	}},
	{Label: "Security and auth", Options: []*Option{
		{ID: "secure-cookies", Label: "Secure cookies", Summary: "Signed and encrypted cookie helpers", Kind: Toggle, Default: []string{"none"}, Choices: toggleChoices("internal/cookies package", "internal/cookies/cookies.go")},
		{ID: "sessions", Label: "Sessions", Summary: "Server-side state per visitor", Kind: Single, Default: []string{"none"}, Choices: []Choice{
			{ID: "none", Label: "None", Summary: "No sessions"},
			{ID: "cookie", Label: "Cookie", Summary: "State in a signed cookie", Files: []string{"internal/sessions/cookie_store.go"}},
			{ID: "database", Label: "Database", Summary: "State in a sessions table", Requires: withDatabase, Files: []string{"internal/sessions/sql_store.go"}},
			{ID: "file", Label: "File", Summary: "State on local disk", Files: []string{"internal/sessions/file_store.go"}},
		}},
		{ID: "basic-auth", Label: "Basic auth", Summary: "HTTP basic auth middleware", Kind: Toggle, Default: []string{"none"}, Choices: toggleChoices("Middleware with credentials from env", "internal/httpx/basic_auth.go")},
	}},
	{Label: "Testing and quality", Options: []*Option{
		{ID: "test-assertions", Label: "Test assertions", Summary: "How tests assert", Kind: Single, Default: []string{"stdlib"}, Choices: []Choice{
			{ID: "stdlib", Label: "Standard library", Summary: "if got != want"},
			{ID: "testify", Label: "testify", Summary: "assert and require"},
		}},
		{ID: "lint-preset", Label: "Lint preset", Summary: "golangci-lint rules", Kind: Single, Default: []string{"standard"}, Choices: []Choice{
			{ID: "none", Label: "None", Summary: "No linter config"},
			{ID: "minimal", Label: "Minimal", Summary: "Bugs only", Files: []string{".golangci.yml"}},
			{ID: "standard", Label: "Standard", Summary: "Bugs, style and gosec", Files: []string{".golangci.yml"}},
			{ID: "strict", Label: "Strict", Summary: "Everything, with thresholds", Files: []string{".golangci.yml"}},
		}},
	}},
	{Label: "Dev loop and project", Options: []*Option{
		{ID: "live-reload", Label: "Live reload", Summary: "Rebuild on save", Kind: Single, Default: []string{"air"}, Choices: []Choice{
			{ID: "none", Label: "None", Summary: "Restart by hand"},
			{ID: "air", Label: "Air", Summary: "go run air", Files: []string{".air.toml"}},
		}},
		{ID: "container", Label: "Container", Summary: "Docker files", Kind: Single, Default: []string{"none"},
			ConditionalDefaults: []ConditionalDefault{{When: Cond{Option: "database", Choices: []string{"postgres", "mysql"}}, Choice: "docker"}},
			Choices: []Choice{
				{ID: "none", Label: "None", Summary: "No container files"},
				{ID: "docker", Label: "Docker", Summary: "Dockerfile and compose.yaml", Files: []string{"Dockerfile", "compose.yaml"}},
			}},
		{ID: "ci", Label: "CI", Summary: "Continuous integration", Kind: Single, Default: []string{"github-actions"}, Choices: []Choice{
			{ID: "none", Label: "None", Summary: "No CI"},
			{ID: "github-actions", Label: "GitHub Actions", Summary: "Lint and test jobs", Files: []string{".github/workflows/ci.yml"}},
		}},
		{ID: "agent-docs", Label: "Agent docs", Summary: "Instructions for coding agents", Kind: Single, Default: []string{"agents-md"}, Choices: []Choice{
			{ID: "none", Label: "None", Summary: "No agent docs"},
			{ID: "agents-md", Label: "AGENTS.md", Summary: "Generated AGENTS.md", Files: []string{"AGENTS.md"}},
		}},
	}},
}

var baseFiles = []string{
	".gitignore", "Makefile", "README.md", "go.mod", "go.sum",
	"cmd/web/main.go", "cmd/web/run.go", "cmd/web/environment.go",
	"internal/bootstrap/handler.go", "internal/config/config.go",
	"internal/health/handler.go", "internal/home/handler.go",
	"internal/httpx/chain.go", "internal/httpx/recover.go", "internal/httpx/serve.go",
	"tests/feature/routes_test.go", "tests/e2e/server_test.go",
}

func optionByID(id string) *Option {
	for _, group := range catalogue {
		for _, option := range group.Options {
			if option.ID == id {
				return option
			}
		}
	}
	return nil
}

func (o *Option) choiceByID(id string) *Choice {
	for i := range o.Choices {
		if o.Choices[i].ID == id {
			return &o.Choices[i]
		}
	}
	return nil
}
