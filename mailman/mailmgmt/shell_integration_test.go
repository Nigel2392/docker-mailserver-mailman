//go:build integration

package mailmgmt_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/djester/testdb"
)

const (
	// Standard docker-mailserver config files
	fileAccounts = "/tmp/docker-mailserver/postfix-accounts.cf"
	fileVirtual  = "/tmp/docker-mailserver/postfix-virtual.cf"
	fileSend     = "/tmp/docker-mailserver/postfix-send-access.cf"
	fileReceive  = "/tmp/docker-mailserver/postfix-receive-access.cf"
)

// TestMain handles the setup and teardown for the entire test suite.
func TestMain(m *testing.M) {
	// Create required directories for the app
	dirs := []string{"./migrations", "./log"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Failed to create dir %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Teardown: Clean up generated directories after tests
	defer func() {
		for _, dir := range dirs {
			os.RemoveAll(dir)
		}
	}()

	// Initialize test database
	_, db := testdb.Open()

	// Bootstrap the Django App with required settings
	app := django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE:             db,
			mailmgmt.MAILSERVER_CONTAINER_NAME: "mailserver", // Required setting for Mailmgmt
		}),
		django.Flag(
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
			django.FlagSkipCmds,
		),
		django.Apps(
			mailmgmt.NewAppConfig, // Bootstraps CONFIG
		),
	)

	// Setup basic stdout logger
	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stderr,
	})

	if err := app.Initialize(); err != nil {
		fmt.Printf("Failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	// Run the tests
	code := m.Run()
	os.Exit(code)
}

// --- HELPER FUNCTIONS ---

// assertInContainerFile uses docker exec to cat a file and check for a substring
func assertInContainerFile(t *testing.T, filename, expectedContent string) {
	t.Helper()
	out, errOut, err := mailmgmt.NewCommand("cat", filename).Exec()
	if err != nil {
		// If the file doesn't exist yet, it's effectively empty.
		// We'll treat a missing file as an error if we EXPECT content.
		t.Fatalf("Failed to read %s: %v\nstderr: %s", filename, err, errOut)
	}
	if !strings.Contains(out, expectedContent) {
		t.Errorf("Expected %q in %s, but it was not found.\nFile content:\n%s", expectedContent, filename, out)
	}
}

// assertNotInContainerFile ensures a specific substring is completely removed
func assertNotInContainerFile(t *testing.T, filename, unexpectedContent string) {
	t.Helper()
	out, _, err := mailmgmt.NewCommand("cat", filename).Exec()
	if err != nil {
		// If file doesn't exist, it definitely doesn't contain the content.
		return
	}
	if strings.Contains(out, unexpectedContent) {
		t.Errorf("Did NOT expect %q in %s, but it was found.\nFile content:\n%s", unexpectedContent, filename, out)
	}
}

// getBaseCommand creates the setup context
func getBaseCommand(t *testing.T) *mailmgmt.SetupCommand {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return mailmgmt.SetupCtx(ctx)
}

// --- TESTS ---

func TestEmailLifecycle(t *testing.T) {
	base := getBaseCommand(t)
	testMail := "test_lifecycle@example.com"
	testPass := "pass123"
	updatePass := "newpass123"

	// Ensure clean state
	_ = base.Email().Delete(testMail)
	assertNotInContainerFile(t, fileAccounts, testMail)

	// Add Email
	err := base.Email().Add(testMail, testPass)
	if err != nil {
		t.Fatalf("Failed to add email: %v", err)
	}
	// Verify actual file insertion
	assertInContainerFile(t, fileAccounts, testMail)

	// List Email
	list, err := base.Email().List(nil)
	if err != nil {
		t.Fatalf("Failed to list emails: %v", err)
	}
	found := false
	for _, addr := range list {
		if addr.Email == testMail {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Email List() parsed successfully, but %s was missing from the structs: %+v", testMail, list)
	}

	// Update Email
	err = base.Email().Update(testMail, updatePass)
	if err != nil {
		t.Fatalf("Failed to update email: %v", err)
	}

	// Delete Email
	err = base.Email().Delete(testMail)
	if err != nil {
		t.Fatalf("Failed to delete email: %v", err)
	}
	// Verify actual file removal
	assertNotInContainerFile(t, fileAccounts, testMail)
}

func TestAliasLifecycle(t *testing.T) {
	base := getBaseCommand(t)
	testAlias := "test_alias_lifecycle@example.com"
	testTarget := "target@example.com"

	// Ensure clean state
	_ = base.Alias().Delete(testAlias, testTarget)
	assertNotInContainerFile(t, fileVirtual, testAlias)

	// Add Alias
	err := base.Alias().Add(testAlias, testTarget)
	if err != nil {
		t.Fatalf("Failed to add alias: %v", err)
	}

	// Format in postfix-virtual.cf is typically "alias target"
	expectedLine := fmt.Sprintf("%s %s", testAlias, testTarget)
	assertInContainerFile(t, fileVirtual, expectedLine)

	// Map Alias
	m, err := base.Alias().List(nil)
	if err != nil {
		t.Fatalf("Failed to map aliases: %v", err)
	}
	found := false
	if aliases, ok := m.Get(testTarget); ok {
		for _, a := range aliases {
			if a == testAlias {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("Alias Map() parsed successfully, but %s -> %s was missing", testAlias, testTarget)
	}

	// Delete Alias
	err = base.Alias().Delete(testAlias, testTarget)
	if err != nil {
		t.Fatalf("Failed to delete alias: %v", err)
	}
	assertNotInContainerFile(t, fileVirtual, testAlias)
}

func TestRestrictLifecycle(t *testing.T) {
	base := getBaseCommand(t)
	testMail := "test_restrict_lifecycle@example.com"

	// Ensure clean state
	_ = base.Restrict().Remove().Send(testMail)
	_ = base.Restrict().Remove().Receive(testMail)

	// Restrict Send
	err := base.Restrict().Add().Send(testMail)
	if err != nil {
		t.Fatalf("Failed to restrict send: %v", err)
	}
	// Verify file insertion
	assertInContainerFile(t, fileSend, fmt.Sprintf("%s REJECT", testMail))

	// Verify List parsing
	sendList, err := base.Restrict().List().Send()
	if err != nil {
		t.Fatalf("Failed to list send restrictions: %v", err)
	}
	found := false
	for _, r := range sendList {
		if r.Address == testMail && r.Status == "REJECT" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Send restriction for %s was missing from List() structs", testMail)
	}

	// Restrict Receive
	err = base.Restrict().Add().Receive(testMail)
	if err != nil {
		t.Fatalf("Failed to restrict receive: %v", err)
	}
	assertInContainerFile(t, fileReceive, fmt.Sprintf("%s REJECT", testMail))

	// Remove Restrictions
	err = base.Restrict().Remove().Send(testMail)
	if err != nil {
		t.Fatalf("Failed to remove send restriction: %v", err)
	}
	assertNotInContainerFile(t, fileSend, testMail)

	err = base.Restrict().Remove().Receive(testMail)
	if err != nil {
		t.Fatalf("Failed to remove receive restriction: %v", err)
	}
	assertNotInContainerFile(t, fileReceive, testMail)
}

func TestQuotaCommand(t *testing.T) {
	base := getBaseCommand(t)
	testMail := "test_quota@example.com"

	// Note: quota add/del doesn't always reflect strictly in a `.cf` file
	// in an easily readable way depending on the DMS version,
	// but we can guarantee the command executes without error.

	// Delete / create email
	base.Email().Delete(testMail)
	err := base.Email().Add(testMail, "pass123")
	if err != nil {
		t.Fatalf("Failed to create quota address: %v", err)
	}

	// Add Quota
	_, _, err = base.Quota().CommandAdd(testMail, "50M").Exec()
	if err != nil {
		t.Fatalf("Failed to set quota: %v", err)
	}

	// Delete Quota
	// (Note: Your CommandDelete implementation takes 'alias' and 'target'
	// but ignores 'alias'. We pass dummy "alias" to fit the signature).
	_, _, err = base.Quota().CommandDelete(testMail).Exec()
	if err != nil {
		t.Fatalf("Failed to delete quota: %v", err)
	}
}
