package mailmgmt_test

import (
	"testing"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
)

func TestMailCommands(t *testing.T) {
	// Base SetupCommand to initiate the chain.
	// Context and Config can be nil for purely testing the argument builder.
	base := mailmgmt.SetupCommand{}

	tests := []struct {
		name     string
		build    func() *mailmgmt.Command
		wantArgs string
		wantErr  string
	}{
		{
			name: "Mail Add",
			build: func() *mailmgmt.Command {
				return base.Email().CommandAdd("test@example.com", "pass123")
			},
			wantArgs: "email add test@example.com pass123",
		},
		{
			name: "Mail Update",
			build: func() *mailmgmt.Command {
				return base.Email().CommandUpdate("test@example.com", "newpass")
			},
			wantArgs: "email update test@example.com newpass",
		},
		{
			name: "Mail Delete - No Emails Provided",
			build: func() *mailmgmt.Command {
				return base.Email().CommandDelete()
			},
			wantErr: "no email adresses provided to delete command",
		},
		{
			name: "Mail Delete - Single Email",
			build: func() *mailmgmt.Command {
				return base.Email().CommandDelete("test@example.com")
			},
			wantArgs: "email del test@example.com",
		},
		{
			name: "Mail Delete - Multiple Emails",
			build: func() *mailmgmt.Command {
				return base.Email().CommandDelete("test1@example.com", "test2@example.com")
			},
			wantArgs: "email del test1@example.com test2@example.com",
		},
		{
			name: "Mail List",
			build: func() *mailmgmt.Command {
				return base.Email().CommandList()
			},
			wantArgs: "email list",
		},
		{
			name: "Restrict Add Send",
			build: func() *mailmgmt.Command {
				return base.Restrict().Add().CommandSend("test@example.com")
			},
			wantArgs: "email restrict add send test@example.com",
		},
		{
			name: "Restrict Add Receive",
			build: func() *mailmgmt.Command {
				return base.Restrict().Add().CommandReceive("test@example.com")
			},
			wantArgs: "email restrict add receive test@example.com",
		},
		{
			name: "Restrict Remove Send",
			build: func() *mailmgmt.Command {
				return base.Restrict().Remove().CommandSend("test@example.com")
			},
			wantArgs: "email restrict rem send test@example.com",
		},
		{
			name: "Restrict Remove Receive",
			build: func() *mailmgmt.Command {
				return base.Restrict().Remove().CommandReceive("test@example.com")
			},
			wantArgs: "email restrict rem receive test@example.com",
		},
		{
			name: "Restrict List Send",
			build: func() *mailmgmt.Command {
				return base.Restrict().List().CommandSend()
			},
			wantArgs: "email restrict list send",
		},
		{
			name: "Restrict List Receive",
			build: func() *mailmgmt.Command {
				return base.Restrict().List().CommandReceive()
			},
			wantArgs: "email restrict list receive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			// 1. Error Validation
			if tt.wantErr != "" {
				if cmd.Error() == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if cmd.Error().Error() != tt.wantErr {
					t.Errorf("got error %q, want %q", cmd.Error().Error(), tt.wantErr)
				}
				// Stop executing further assertions if an error was expected
				return
			} else if cmd.Error() != nil {
				t.Fatalf("unexpected error: %v", cmd.Error())
			}

			// 2. Arguments Validation
			gotArgs := cmd.String()
			if gotArgs != tt.wantArgs {
				t.Errorf("got args %q, want %q", gotArgs, tt.wantArgs)
			}
		})
	}
}
