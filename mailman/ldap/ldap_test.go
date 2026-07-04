package ldap_test

import (
	"os"
	"testing"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/quest"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/goldcrest"

	appldap "github.com/Nigel2392/docker-mailserver-mailman/mailman/ldap"
	"github.com/Nigel2392/go-django/src/djester/testdb"

	"github.com/go-ldap/ldap/v3"
)

func TestLDAPServer(t *testing.T) {
	var app = django.App(
		django.Configure(map[string]interface{}{
			appldap.APPVAR_LDAP_PORT:  "33890",
			appldap.DEFAULT_LDAP_HOST: "127.0.0.1",
			django.APPVAR_DATABASE: func() drivers.Database {
				var _, db = testdb.Open()
				return db
			}(),
		}),
		django.AppLogger(&logger.Logger{
			Level:       logger.INF,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stdout,
		}),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			appldap.NewAppConfig(),
		),
		django.Flag(
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
			django.FlagSkipCmds,
		),
	)

	// create tables
	var tables = quest.Table(t,
		&auth.User{},
		&users.Group{},
		&users.Permission{},
		&users.UserGroup{},
		&users.GroupPermission{},
		&users.UserPermission{},
		&appldap.Domain{},
		&appldap.MailAlias{},
		&appldap.MailAliasUser{},
	)

	// Reset the definitions to ensure all models are registered
	// before reverse fields are fully setup.
	attrs.ResetDefinitions.Send(nil)

	tables.Create()
	defer tables.Drop()

	for _, fn := range goldcrest.Get[django.DjangoHook](django.HOOK_SERVER_STARTUP) {
		t.Log("starting up...")
		if err := fn(app); err != nil {
			t.Fatalf("failed to startup: %v", err)
		}
	}

	// Create a test user
	testUser := &auth.User{
		Username:  "testuser",
		Email:     drivers.MustParseEmail("testuser@example.com"),
		Password:  auth.NewPassword("supersecret"),
		FirstName: "TestFirst",
		LastName:  "TestLast",
		Base: users.Base{
			IsActive:        true,
			IsAdministrator: true,
		},
	}

	if _, err := auth.GetUserQuerySet().Create(testUser); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create a test alias
	testAlias := models.Setup(&appldap.MailAlias{
		Source:   drivers.MustParseEmail("info@example.com"),
		IsActive: true,
	})

	if _, err := queries.GetQuerySet(&appldap.MailAlias{}).Create(testAlias); err != nil {
		t.Fatalf("Failed to create test alias: %v", err)
	}

	var domain = &appldap.Domain{
		Name:   "Default Domain",
		Domain: "example.com",
	}
	if err := queries.CreateObject(domain); err != nil {
		t.Fatalf("Failed to create test alias: %v", err)
	}

	if _, err := testAlias.Destination.Objects().AddTarget(testUser); err != nil {
		t.Fatalf("Failed to create test alias: %v", err)
	}

	// Give the server a tiny fraction of a second to bind the TCP port
	time.Sleep(1 * time.Second)

	// Ensure server shuts down after tests complete
	defer func() {
		t.Log("shutting down...")
		for _, fn := range goldcrest.Get[django.DjangoHook](django.HOOK_SERVER_SHUTDOWN) {
			if err := fn(app); err != nil {
				t.Fatalf("failed to shutdown: %v", err)
			}
		}
	}()

	t.Run("Valid Authentication (BIND)", func(t *testing.T) {
		l, err := ldap.DialURL("ldap://127.0.0.1:33890")
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer l.Close()
		defer l.Unbind()

		// Attempt to login with our test user credentials
		err = l.Bind("mail=testuser@example.com", "supersecret")
		if err != nil {
			t.Errorf("Expected successful bind, got error: %v", err)
		}
	})

	t.Run("Invalid Authentication (BIND)", func(t *testing.T) {
		l, err := ldap.DialURL("ldap://127.0.0.1:33890")
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer l.Close()
		defer l.Unbind()

		// Attempt to login with bad password
		err = l.Bind("mail=testuser@example.com", "wrongpassword")
		if err == nil {
			t.Errorf("Expected bind to fail with wrong password, but it succeeded")
		} else if !ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			t.Errorf("Expected InvalidCredentials error, got: %v", err)
		}
	})

	t.Run("Search for Existing User", func(t *testing.T) {
		l, _ := ldap.DialURL("ldap://127.0.0.1:33890")
		defer l.Close()
		defer l.Unbind()
		l.Bind("mail=testuser@example.com", "supersecret") // Authenticate the session first!

		// Build the search request exactly how Postfix would send it
		searchReq := ldap.NewSearchRequest(
			"dc=mydomain,dc=loc", // Base DN
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			"(&(objectClass=user)(mail=testuser@example.com))", // The Filter
			[]string{"uid", "mail", "cn"},                      // Attributes we want back
			nil,
		)

		res, err := l.Search(searchReq)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(res.Entries) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(res.Entries))
		}

		entry := res.Entries[0]
		if entry.GetAttributeValue("mail") != "testuser@example.com" {
			t.Errorf("Expected mail 'testuser@example.com', got %s", entry.GetAttributeValue("mail"))
		}
		if entry.GetAttributeValue("cn") != "TestFirst TestLast" {
			t.Errorf("Expected cn 'TestFirst TestLast', got %s", entry.GetAttributeValue("cn"))
		}
	})

	t.Run("Search for Existing Alias", func(t *testing.T) {
		l, _ := ldap.DialURL("ldap://127.0.0.1:33890")
		defer l.Close()
		defer l.Unbind()
		l.Bind("mail=testuser@example.com", "supersecret")

		searchReq := ldap.NewSearchRequest(
			"dc=mydomain,dc=loc",
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			"(&(objectClass=user)(otherMailbox=info@example.com))",
			[]string{"mail", "otherMailbox"},
			nil,
		)

		res, err := l.Search(searchReq)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(res.Entries) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(res.Entries))
		}

		// Verify Postfix will be told to route to 'testuser@example.com'
		targetMail := res.Entries[0].GetAttributeValue("mail")
		if targetMail != "testuser@example.com" {
			t.Errorf("Expected alias to resolve to 'testuser@example.com', got %s", targetMail)
		}
	})
}
