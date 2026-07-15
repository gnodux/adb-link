package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
)

// cliCommand is a single CLI subcommand handler.
type cliCommand struct {
	Name        string
	Synopsis    string
	Description string
	Run         func(args []string, stdout, stderr io.Writer) int
}

var cliCommands = []cliCommand{
	{Name: "list-datasources", Synopsis: "list-datasources [--json]", Description: "List configured datasources", Run: cliListDatasources},
	{Name: "list-databases", Synopsis: "list-databases <datasource> [--json]", Description: "List databases in a datasource", Run: cliListDatabases},
	{Name: "list-tables", Synopsis: "list-tables <datasource> <database> [--json]", Description: "List tables in a database", Run: cliListTables},
	{Name: "describe", Synopsis: "describe <datasource> <database> <table> [--json]", Description: "Show table schema", Run: cliDescribe},
	{Name: "query", Synopsis: "query <datasource> <sql> [--database NAME] [--limit N] [--timeout SECS] [--json]", Description: "Execute a SQL query", Run: cliQuery},
	{Name: "ping", Synopsis: "ping <datasource>", Description: "Test datasource connectivity", Run: cliPing},
}

func cliUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: adb-link cli <command> [arguments]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, c := range cliCommands {
		fmt.Fprintf(w, "  %-90s %s\n", c.Synopsis, c.Description)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Global flags:")
	fmt.Fprintln(w, "  --json    Output as JSON instead of human-readable table")
}

// runCLI dispatches to a CLI subcommand.
func runCLI() {
	stdout, stderr := os.Stdout, os.Stderr
	args := os.Args[2:]
	if len(args) == 0 {
		cliUsage(stderr)
		os.Exit(2)
	}
	cmd := args[0]
	for _, c := range cliCommands {
		if c.Name == cmd {
			os.Exit(c.Run(args[1:], stdout, stderr))
		}
	}
	fmt.Fprintf(stderr, "unknown cli command: %s\n\n", cmd)
	cliUsage(stderr)
	os.Exit(2)
}

type cliFlags struct {
	JSON     bool
	Database string
	Limit    int
	Timeout  int
}

func parseFlags(args []string) (cliFlags, []string, error) {
	f := cliFlags{Limit: 1000, Timeout: 30}
	fs := flag.NewFlagSet("cli", flag.ContinueOnError)
	fs.BoolVar(&f.JSON, "json", false, "output as JSON")
	fs.StringVar(&f.Database, "database", "", "target database (for query)")
	fs.IntVar(&f.Limit, "limit", 1000, "max rows to return (query)")
	fs.IntVar(&f.Timeout, "timeout", 30, "query timeout in seconds")
	if err := fs.Parse(args); err != nil {
		return f, nil, err
	}
	return f, fs.Args(), nil
}

// setupContainer creates a service container without starting background workers.
func setupContainer() *services.Container {
	settings := config.DefaultSettings()
	return services.NewContainer(settings)
}

func failf(w io.Writer, format string, args ...any) int {
	fmt.Fprintf(w, format+"\n", args...)
	return 1
}

func cliListDatasources(args []string, stdout, stderr io.Writer) int {
	flags, positional, err := parseFlags(args)
	if err != nil {
		return failf(stderr, "error: %s", err)
	}
	if len(positional) != 0 {
		return failf(stderr, "usage: adb-link cli list-datasources [--json]")
	}

	c := setupContainer()
	defer c.Stop()

	infos := c.ConfigService.ListDatasources()
	if flags.JSON {
		return writeJSON(stdout, infos)
	}

	if len(infos) == 0 {
		fmt.Fprintln(stdout, "No datasources configured.")
		return 0
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tDESCRIPTION\tSHADOW")
	for _, i := range infos {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%v\n", i.Name, i.Type, i.Description, i.Shadow)
	}
	tw.Flush()
	return 0
}

func cliListDatabases(args []string, stdout, stderr io.Writer) int {
	flags, positional, err := parseFlags(args)
	if err != nil {
		return failf(stderr, "error: %s", err)
	}
	if len(positional) != 1 {
		return failf(stderr, "usage: adb-link cli list-databases <datasource> [--json]")
	}
	ds := positional[0]

	c := setupContainer()
	defer c.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flags.Timeout)*time.Second)
	defer cancel()

	dbs, err := c.SchemaService.GetDatabases(ctx, ds, "")
	if err != nil {
		return failf(stderr, "error: %s", err)
	}

	if flags.JSON {
		return writeJSON(stdout, dbs)
	}
	if len(dbs) == 0 {
		fmt.Fprintln(stdout, "No databases found.")
		return 0
	}
	for _, d := range dbs {
		if d.Comment != "" {
			fmt.Fprintf(stdout, "%s\t%s\n", d.Name, d.Comment)
		} else {
			fmt.Fprintln(stdout, d.Name)
		}
	}
	return 0
}

func cliListTables(args []string, stdout, stderr io.Writer) int {
	flags, positional, err := parseFlags(args)
	if err != nil {
		return failf(stderr, "error: %s", err)
	}
	if len(positional) != 2 {
		return failf(stderr, "usage: adb-link cli list-tables <datasource> <database> [--json]")
	}
	ds, db := positional[0], positional[1]

	c := setupContainer()
	defer c.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flags.Timeout)*time.Second)
	defer cancel()

	schema, err := c.SchemaService.GetSchema(ctx, ds, db, "")
	if err != nil {
		return failf(stderr, "error: %s", err)
	}

	if flags.JSON {
		return writeJSON(stdout, schema)
	}

	if len(schema.Tables) == 0 && len(schema.Views) == 0 {
		fmt.Fprintln(stdout, "No tables or views found.")
		return 0
	}
	if len(schema.Tables) > 0 {
		fmt.Fprintln(stdout, "TABLES:")
		for _, t := range schema.Tables {
			comment := ""
			if t.Comment != nil && *t.Comment != "" {
				comment = "\t" + *t.Comment
			}
			fmt.Fprintf(stdout, "  %s%s\n", t.Name, comment)
		}
	}
	if len(schema.Views) > 0 {
		fmt.Fprintln(stdout, "VIEWS:")
		for _, v := range schema.Views {
			comment := ""
			if v.Comment != nil && *v.Comment != "" {
				comment = "\t" + *v.Comment
			}
			fmt.Fprintf(stdout, "  %s%s\n", v.Name, comment)
		}
	}
	return 0
}

func cliDescribe(args []string, stdout, stderr io.Writer) int {
	flags, positional, err := parseFlags(args)
	if err != nil {
		return failf(stderr, "error: %s", err)
	}
	if len(positional) != 3 {
		return failf(stderr, "usage: adb-link cli describe <datasource> <database> <table> [--json]")
	}
	ds, db, tbl := positional[0], positional[1], positional[2]

	c := setupContainer()
	defer c.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flags.Timeout)*time.Second)
	defer cancel()

	info, err := c.SchemaService.GetTableInfo(ctx, ds, db, tbl, "")
	if err != nil {
		vinfo, verr := c.SchemaService.GetViewInfo(ctx, ds, db, tbl, "")
		if verr != nil {
			return failf(stderr, "error: %s", err)
		}
		info = vinfo
	}

	if flags.JSON {
		return writeJSON(stdout, info)
	}

	fmt.Fprintf(stdout, "Table: %s\n", info.Name)
	if info.SchemaName != nil && *info.SchemaName != "" {
		fmt.Fprintf(stdout, "Schema: %s\n", *info.SchemaName)
	}
	if info.Comment != nil && *info.Comment != "" {
		fmt.Fprintf(stdout, "Comment: %s\n", *info.Comment)
	}
	if len(info.Columns) == 0 {
		fmt.Fprintln(stdout, "No columns.")
		return 0
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COLUMN\tTYPE\tNULLABLE\tDEFAULT\tPK\tCOMMENT")
	for _, col := range info.Columns {
		def := ""
		if col.Default != nil {
			def = *col.Default
		}
		comment := ""
		if col.Comment != nil {
			comment = *col.Comment
		}
		pk := ""
		if col.IsPrimaryKey {
			pk = "YES"
		}
		fmt.Fprintf(tw, "%s\t%s\t%v\t%s\t%s\t%s\n", col.Name, col.Type, col.Nullable, def, pk, comment)
	}
	tw.Flush()
	return 0
}

func cliQuery(args []string, stdout, stderr io.Writer) int {
	flags, positional, err := parseFlags(args)
	if err != nil {
		return failf(stderr, "error: %s", err)
	}
	if len(positional) < 2 {
		return failf(stderr, "usage: adb-link cli query <datasource> <sql...> [--database NAME] [--limit N] [--timeout SECS] [--json]")
	}
	ds, sqlStr := positional[0], strings.Join(positional[1:], " ")

	c := setupContainer()
	defer c.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flags.Timeout)*time.Second)
	defer cancel()

	req := &models.QueryRequest{
		DatasourceName: ds,
		Database:       flags.Database,
		SQL:            sqlStr,
		Limit:          flags.Limit,
		TimeoutSeconds: flags.Timeout,
	}
	result, err := c.QueryService.Execute(ctx, req, "")
	if err != nil {
		return failf(stderr, "error: %s", err)
	}

	if flags.JSON {
		return writeJSON(stdout, result)
	}
	return printQueryResult(stdout, result)
}

func cliPing(args []string, stdout, stderr io.Writer) int {
	_, positional, err := parseFlags(args)
	if err != nil {
		return failf(stderr, "error: %s", err)
	}
	if len(positional) != 1 {
		return failf(stderr, "usage: adb-link cli ping <datasource>")
	}
	ds := positional[0]

	c := setupContainer()
	defer c.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	info, err := c.ConnectionService.GetServerInfo(ctx, ds); if err != nil {
		return failf(stderr, "ping failed: %s", err)
	}
	fmt.Fprintf(stdout, "OK (version=%s, %s)\n", info.Version, time.Since(start).Round(time.Millisecond))
	return 0
}

func printQueryResult(w io.Writer, r *models.QueryResult) int {
	if len(r.Columns) == 0 {
		fmt.Fprintf(w, "Query OK, %d row(s) affected in %.2f ms\n", r.RowCount, r.ExecutionTimeMs)
		return 0
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, c := range r.Columns {
		fmt.Fprintf(tw, "%s\t", c.Name)
	}
	fmt.Fprintln(tw)
	for _, row := range r.Rows {
		for i, v := range row {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, formatValue(v))
		}
		fmt.Fprintln(tw)
	}
	tw.Flush()
	fmt.Fprintf(w, "\n%d row(s) in %.2f ms", r.RowCount, r.ExecutionTimeMs)
	if r.Truncated {
		fmt.Fprintf(w, " (truncated to limit %d)", r.Limit)
	}
	fmt.Fprintln(w)
	return 0
}

func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	case []byte:
		return string(x)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func writeJSON(w io.Writer, v any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "json encode error: %s\n", err)
		return 1
	}
	return 0
}
