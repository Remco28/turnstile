package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Remco28/turnstile/internal/config"
	"github.com/Remco28/turnstile/internal/httpapi"
	"github.com/Remco28/turnstile/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	cfg := config.Load()
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()

	switch args[0] {
	case "create-user":
		if len(args) < 2 {
			return errors.New("usage: turnstile create-user <name>")
		}
		user, err := st.CreateUser(args[1])
		if err != nil {
			return err
		}
		return printJSON(user)
	case "create-project":
		return createProjectCommand(st, args[1:])
	case "create-token":
		return createTokenCommand(st, args[1:])
	case "list-users":
		users, err := st.ListUsers()
		if err != nil {
			return err
		}
		return printJSON(users)
	case "list-projects":
		projects, err := st.ListProjects()
		if err != nil {
			return err
		}
		return printJSON(projects)
	case "list-tokens":
		return listTokensCommand(st, args[1:])
	case "show-token":
		return showTokenCommand(st, args[1:])
	case "reissue-token":
		return reissueTokenCommand(st, args[1:])
	case "replace-grants":
		return replaceGrantsCommand(st, args[1:])
	case "who-has-access":
		return whoHasAccessCommand(st, args[1:])
	case "access-log":
		return accessLogCommand(st, args[1:])
	case "revoke-token":
		return revokeTokenCommand(st, args[1:])
	case "validate":
		return validateCommand(st, args[1:])
	case "serve":
		return serveCommand(cfg, st, args[1:])
	default:
		return usageError()
	}
}

func createProjectCommand(st *store.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: turnstile create-project <slug> [--description ...]")
	}
	slug := ""
	parseArgs := args
	if !strings.HasPrefix(args[0], "-") {
		slug = args[0]
		parseArgs = args[1:]
	}
	fs := flag.NewFlagSet("create-project", flag.ContinueOnError)
	description := fs.String("description", "", "project description")
	if err := fs.Parse(parseArgs); err != nil {
		return err
	}
	if slug == "" {
		if fs.NArg() != 1 {
			return errors.New("usage: turnstile create-project <slug> [--description ...]")
		}
		slug = fs.Arg(0)
	}
	project, err := st.CreateProject(slug, *description)
	if err != nil {
		return err
	}
	return printJSON(project)
}

func createTokenCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("create-token", flag.ContinueOnError)
	user := fs.String("user", "", "user name")
	label := fs.String("label", "", "token label")
	var projects multiString
	fs.Var(&projects, "project", "project slug (repeatable)")
	expiresIn := fs.String("expires-in", "", "Go duration like 720h")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var expiresAt *time.Time
	if *expiresIn != "" {
		duration, err := time.ParseDuration(*expiresIn)
		if err != nil {
			return fmt.Errorf("invalid expires-in: %w", err)
		}
		value := time.Now().UTC().Add(duration)
		expiresAt = &value
	}
	record, err := st.CreateToken(*user, projects, *label, expiresAt)
	if err != nil {
		return err
	}
	return printJSON(record)
}

func listTokensCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("list-tokens", flag.ContinueOnError)
	user := fs.String("user", "", "filter by user")
	if err := fs.Parse(args); err != nil {
		return err
	}
	items, err := st.ListTokens(*user)
	if err != nil {
		return err
	}
	return printJSON(items)
}

func showTokenCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("show-token", flag.ContinueOnError)
	id := fs.Int64("id", 0, "token id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == 0 {
		return errors.New("usage: turnstile show-token --id <id>")
	}
	record, err := st.GetToken(*id)
	if err != nil {
		return err
	}
	return printJSON(record)
}

func reissueTokenCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("reissue-token", flag.ContinueOnError)
	id := fs.Int64("id", 0, "token id")
	label := fs.String("label", "", "new token label (optional)")
	expiresIn := fs.String("expires-in", "", "Go duration like 720h")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == 0 {
		return errors.New("usage: turnstile reissue-token --id <id> [--label ...] [--expires-in 720h]")
	}
	var expiresAt *time.Time
	if *expiresIn != "" {
		duration, err := time.ParseDuration(*expiresIn)
		if err != nil {
			return fmt.Errorf("invalid expires-in: %w", err)
		}
		value := time.Now().UTC().Add(duration)
		expiresAt = &value
	}
	result, err := st.ReissueToken(*id, *label, expiresAt)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func replaceGrantsCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("replace-grants", flag.ContinueOnError)
	id := fs.Int64("id", 0, "token id")
	var projects multiString
	fs.Var(&projects, "project", "project slug (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == 0 || len(projects) == 0 {
		return errors.New("usage: turnstile replace-grants --id <id> --project <slug> [--project <slug> ...]")
	}
	record, err := st.ReplaceTokenProjects(*id, projects)
	if err != nil {
		return err
	}
	return printJSON(record)
}

func whoHasAccessCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("who-has-access", flag.ContinueOnError)
	project := fs.String("project", "", "project slug")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *project == "" {
		return errors.New("usage: turnstile who-has-access --project <slug>")
	}
	items, err := st.ListProjectAccess(*project)
	if err != nil {
		return err
	}
	return printJSON(items)
}

func accessLogCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("access-log", flag.ContinueOnError)
	project := fs.String("project", "", "project slug")
	limit := fs.Int("limit", 20, "max rows")
	if err := fs.Parse(args); err != nil {
		return err
	}
	items, err := st.ListAccessLog(*project, *limit)
	if err != nil {
		return err
	}
	return printJSON(items)
}

func revokeTokenCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("revoke-token", flag.ContinueOnError)
	id := fs.Int64("id", 0, "token id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == 0 {
		return errors.New("usage: turnstile revoke-token --id <id>")
	}
	record, err := st.RevokeToken(*id)
	if err != nil {
		return err
	}
	return printJSON(record)
}

func validateCommand(st *store.Store, args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	tokenValue := fs.String("token", "", "raw token")
	project := fs.String("project", "", "project slug")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := st.ValidateToken(*tokenValue, *project, "cli", "turnstile-cli")
	if err != nil {
		return err
	}
	return printJSON(result)
}

func serveCommand(cfg config.Config, st *store.Store, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := fs.String("listen", cfg.ListenAddr, "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	server := &http.Server{
		Addr:    *listen,
		Handler: httpapi.New(st).Handler(),
	}
	log.Printf("turnstile listening on %s", *listen)
	return server.ListenAndServe()
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func usageError() error {
	return errors.New("usage: turnstile <create-user|create-project|create-token|list-users|list-projects|list-tokens|show-token|reissue-token|replace-grants|who-has-access|access-log|revoke-token|validate|serve>")
}

type multiString []string

func (m *multiString) String() string { return fmt.Sprintf("%v", []string(*m)) }

func (m *multiString) Set(value string) error {
	*m = append(*m, value)
	return nil
}

var _ = strconv.IntSize
