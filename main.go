package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"

	_ "github.com/lib/pq"
)

var pluginName = "krakend-server-example"
var HandlerRegisterer = registerer(pluginName)
var logger Logger = noopLogger{}

type registerer string
type Logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warning(v ...interface{})
	Error(v ...interface{})
	Critical(v ...interface{})
	Fatal(v ...interface{})
}

type noopLogger struct{}

func (n noopLogger) Debug(_ ...interface{})    {}
func (n noopLogger) Info(_ ...interface{})     {}
func (n noopLogger) Warning(_ ...interface{})  {}
func (n noopLogger) Error(_ ...interface{})    {}
func (n noopLogger) Critical(_ ...interface{}) {}
func (n noopLogger) Fatal(_ ...interface{})    {}

func (r registerer) RegisterHandlers(f func(
	name string,
	handler func(context.Context, map[string]interface{}, http.Handler) (http.Handler, error),
)) {
	f(string(r), r.registerHandlers)
}

func (r registerer) registerHandlers(_ context.Context, extra map[string]interface{}, h http.Handler) (http.Handler, error) {
	config, ok := extra[pluginName].(map[string]interface{})
	if !ok {
		return h, errors.New("configuration not found")
	}

	path, _ := config["path"].(string)
	schema, _ := config["schema"].(string)
	table, _ := config["table"].(string)
	dbHost, _ := config["db_host"].(string)
	dbPort, _ := config["db_port"].(float64) // JSON'da tüm sayılar float olarak okunur
	dbUser, _ := config["db_user"].(string)
	dbPass, _ := config["db_pass"].(string)
	dbName, _ := config["db_name"].(string)
	fields, _ := config["fields"].(map[string]interface{})

	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbHost, int(dbPort), dbUser, dbPass, dbName)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		logger.Error(err)
		return h, err
	}
	defer db.Close()

	columns := make([]string, 0, len(fields))
	values := make([]string, 0, len(fields))
	args := make([]interface{}, 0, len(fields))
	for col, val := range fields {
		columns = append(columns, col)
		values = append(values, "?")
		args = append(args, val)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s.%s (%s) VALUES (%s)",
		schema,
		table,
		strings.Join(columns, ", "),
		strings.Join(values, ", "),
	)

	_, err = db.Exec(query, args...)
	if err != nil {
		logger.Error(err)
		return h, err
	}

	logger.Debug(fmt.Sprintf("The plugin is now hijacking the path %s", path))

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		if req.URL.Path != path {
			h.ServeHTTP(w, req)
			return
		}

		fmt.Fprintf(w, "Hello, %q", html.EscapeString(req.URL.Path))
		logger.Debug("request:", html.EscapeString(req.URL.Path))
	}), nil
}

func main() {}

func (registerer) RegisterLogger(v interface{}) {
	l, ok := v.(Logger)
	if !ok {
		return
	}
	logger = l
	logger.Debug(fmt.Sprintf("[PLUGIN: %s] Logger loaded", HandlerRegisterer))
}
