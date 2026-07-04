package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/ldap"
	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
	"github.com/Nigel2392/docker-mailserver-mailman/mailman/sieve"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/goldcrest"

	// "github.com/Nigel2392/go-django/src/contrib/translations"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/logger"
)

const APP_SHUTDOWN = "APP_SHUTDOWN"

func GetEnv(key string, default_ ...string) string {
	var val, ok = os.LookupEnv(key)
	if !ok && len(default_) > 0 {
		return default_[0]
	}
	return val
}

func GetEnvT[T any](key string, default_ T, convert func(in string) (out T, err error)) T {
	var val, ok = os.LookupEnv(key)
	if !ok {
		return default_
	}

	v, err := convert(val)
	if err != nil {
		return default_
	}

	return v
}

func main() {
	var (
		RUNNING_IN_DOCKER = GetEnvT("DOCKER", false, strconv.ParseBool)

		MAILSERVER_CONTAINER_NAME  = GetEnv("MAILSERVER_CONTAINER_NAME")
		MAILSERVER_CACHING_ENABLED = GetEnvT("MAILSERVER_CACHING_ENABLED", true, strconv.ParseBool)

		MAILMAN_INTERFACE = GetEnv("MAILMAN_INTERFACE", "127.0.0.1")
		MAILMAN_PORT      = GetEnv("MAILMAN_PORT", "8080")

		MAILMAN_SQLITE_DB = GetEnv("MAILMAN_SQLITE_DB", "./db/sqlite.db")

		MAILMAN_LOG       = GetEnv("MAILMAN_LOG", "./log/mailman.log")
		MAILMAN_ERROR_LOG = GetEnv("MAILMAN_ERROR_LOG", "./log/mailman.error.log")

		MAILMAN_SIEVE_TEMPLATE = GetEnv(
			"MAILMAN_SIEVE_TEMPLATE",
			"./templates/tmp/docker-mailserver/before.dovecot.sieve.tmpl",
		)

		MAILMAN_ADMIN_EMAIL = GetEnv("MAILMAN_ADMIN_EMAIL")    //, "Administrator@example.com")
		MAILMAN_ADMIN_PASS  = GetEnv("MAILMAN_ADMIN_PASSWORD") //, "Your-real-db-password!123")
	)

	var files = make(map[string]*os.File)
	for _, logPath := range []string{MAILMAN_LOG, MAILMAN_ERROR_LOG} {
		if _, ok := files[logPath]; ok {
			continue
		}

		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}

		files[logPath] = f
	}

	db, err := drivers.Open(context.Background(), "sqlite3", MAILMAN_SQLITE_DB)
	if err != nil {
		panic(err)
	}

	var app = django.App(
		django.AppSettings(django.Config(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS:         []string{"*"},
			django.APPVAR_DATABASE:              db,
			django.APPVAR_HOST:                  MAILMAN_INTERFACE,
			django.APPVAR_PORT:                  MAILMAN_PORT,
			django.APPVAR_DEBUG:                 !RUNNING_IN_DOCKER,
			django.APPVAR_RECOVERER:             false,
			auth.APPVAR_AUTH_EMAIL_LOGIN:        true,
			auth.APPVAR_ALLOW_USER_REGISTER:     false,
			migrator.APPVAR_MIGRATION_DIR:       "./migrations",
			mailmgmt.MAILSERVER_CONTAINER_NAME:  MAILSERVER_CONTAINER_NAME,
			mailmgmt.MAILSERVER_CACHING_ENABLED: MAILSERVER_CACHING_ENABLED,
			sieve.MAILMAN_SIEVE_TEMPLATE:        MAILMAN_SIEVE_TEMPLATE,
		})),
		django.AppLogger(&logger.Logger{
			Level:       logger.DBG,
			OutputTime:  true,
			OutputDebug: io.MultiWriter(os.Stdout, files[MAILMAN_LOG]),
			OutputInfo:  io.MultiWriter(os.Stdout, files[MAILMAN_LOG]),
			OutputWarn:  io.MultiWriter(os.Stdout, files[MAILMAN_LOG]),
			OutputError: io.MultiWriter(os.Stderr, files[MAILMAN_ERROR_LOG]),
		}),
		django.Apps(
			migrator.NewAppConfig,
			session.NewAppConfig,
			auth.NewAppConfig,
			mailmgmt.NewAppConfig,
			sieve.NewAppConfig,
			ldap.NewAppConfig,
			// translate.NewAppConfig,
			messages.NewAppConfig,
			NewAppConfig,
		),
	)

	var interrupts = make(chan os.Signal, 1)
	signal.Notify(interrupts, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-interrupts

		var exitCode int
		for _, fn := range django.ConfigGet(django.Global.Settings, APP_SHUTDOWN, []func() error{}) {
			if err := fn(); err != nil {
				fmt.Printf("Error while executing shutdown function: %v\n", err)
				exitCode = 1
			}
		}

		if err := django.Global.Quit(); err != nil {
			fmt.Printf("failed to close django app: %v\n", err)
			exitCode = 1
		}

		os.Exit(exitCode)
	}()

	goldcrest.Register(django.HOOK_SERVER_SHUTDOWN, 0, django.DjangoHook(func(a *django.Application) error {
		var errs = make([]error, 0)
		for _, file := range files {
			err := file.Close()
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"failed to close log file %s: %w\n", file.Name(), err,
				))
			}
		}

		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf(
				"failed to close database file %q: %w\n", MAILMAN_SQLITE_DB, err,
			))
		}

		return errors.Join(errs...)
	}))

	if !RUNNING_IN_DOCKER {
		logger.Warn("Executing in DEBUG mode.")
	}

	checks.Shutup("migrator.engine.too_many_migrations", true)
	checks.Shutup("migrator.engine.needs_makemigrations", true)

	// Initialize Go-Django before doing anything with the database.
	err = app.Initialize()
	if err != nil {
		panic(err)
	}

	if _, _, err := migrator.AutoMigrate(context.Background()); err != nil {
		panic(err)
	}

	// testCommands()
	//
	// os.Exit(0)

	userCount, err := queries.CountObjects(&auth.User{})
	if err != nil {
		panic(fmt.Errorf("failed to count previously existing users: %w", err))
	}

	// If admin credentials are provided in the environment, use them
	// to create a default admin account (if none existed previously)
	// This will only be done if no users exist in the database yet.
	switch {
	case userCount == 0 && MAILMAN_ADMIN_EMAIL != "" && MAILMAN_ADMIN_PASS != "":
		var user = &auth.User{}
		var e, _ = mail.ParseAddress(MAILMAN_ADMIN_EMAIL)
		user.Email = (*drivers.Email)(e)
		user.Username = "admin"
		user.IsAdministrator = true
		user.IsActive = true
		user.Password = auth.NewPassword(MAILMAN_ADMIN_PASS)

		if user, err = queries.GetQuerySet(&auth.User{}).Filter("Email", e.Address).Create(user); err != nil {
			panic(fmt.Errorf("failed to create admin user: %w", err))
		}

		logger.Infof("Admin user created: %v %s %s %t %t", user.ID, user.Username, user.Email, user.IsAdministrator, user.IsActive)

	case userCount > 0 && MAILMAN_ADMIN_EMAIL != "" && MAILMAN_ADMIN_PASS != "":
		logger.Warnf("%d users already exist, but \"MAILMAN_ADMIN_EMAIL\" and \"MAILMAN_ADMIN_PASS\" are still set.", userCount)
	case userCount == 0 && MAILMAN_ADMIN_EMAIL == "" && MAILMAN_ADMIN_PASS == "":
		logger.Warnf("%d users exist, but no admin account is configured", userCount)
	default:
		logger.Infof("%d users in database.", userCount)
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
