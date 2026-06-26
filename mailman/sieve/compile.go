package sieve

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/moby/moby/client"
)

type SieveConfigData struct {
	BannedEmails []*BannedEmail
	Forwards     []*ForwardedEmail
	Vacations    []*VacationRule
}

func Query(ctx context.Context) (*SieveConfigData, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	bannedEmailRows, err := queries.GetQuerySet(&BannedEmail{}).Select("Email", "Action").All()
	if err != nil {
		return nil, err
	}

	forwardedRows, err := queries.GetQuerySet(&ForwardedEmail{}).Select("Source", "Destination", "KeepCopy").All()
	if err != nil {
		return nil, err
	}

	vacationRuleRows, err := queries.GetQuerySet(&VacationRule{}).Select("ForEmail", "Enabled", "Days", "Subject", "Message").All()
	if err != nil {
		return nil, err
	}

	var config = &SieveConfigData{
		BannedEmails: make([]*BannedEmail, 0, len(bannedEmailRows)),
		Forwards:     make([]*ForwardedEmail, 0, len(forwardedRows)),
		Vacations:    make([]*VacationRule, 0, len(vacationRuleRows)),
	}
	for i := 0; i < max(len(bannedEmailRows), len(forwardedRows), len(vacationRuleRows)); i++ {
		if i < len(bannedEmailRows) {
			config.BannedEmails = append(config.BannedEmails, bannedEmailRows[i].Object)
		}
		if i < len(forwardedRows) {
			config.Forwards = append(config.Forwards, forwardedRows[i].Object)
		}
		if i < len(vacationRuleRows) {
			config.Vacations = append(config.Vacations, vacationRuleRows[i].Object)
		}
	}

	return config, nil
}

func Compile(ctx context.Context, config *SieveConfigData, tarDest io.Writer) error {
	if !_app._enabled {
		return ErrAppNotEnabled
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if config == nil {
		panic("cannot compile nil sieve config")
	}

	tmpl, err := template.New("sieve").ParseFiles(django.ConfigGet[string](
		django.Global.Settings, MAILMAN_SIEVE_TEMPLATE,
	))
	if err != nil {
		return fmt.Errorf("failed to parse sieve template: %w", err)
	}

	var scriptBuffer bytes.Buffer
	if err := tmpl.Execute(&scriptBuffer, config); err != nil {
		return fmt.Errorf("failed to execute sieve template: %w", err)
	}

	tarWriter := tar.NewWriter(tarDest)
	scriptBytes := scriptBuffer.Bytes()
	header := &tar.Header{
		Name:    "before.dovecot.sieve",
		Mode:    0644,
		Size:    int64(len(scriptBytes)),
		ModTime: time.Now(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}
	if _, err := tarWriter.Write(scriptBytes); err != nil {
		return fmt.Errorf("failed to write tar body: %w", err)
	}
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	return nil
}

func Upload(ctx context.Context, config *SieveConfigData) error {
	if !_app._enabled {
		return ErrAppNotEnabled
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var b = bytes.Buffer{}
	if err := Compile(ctx, config, &b); err != nil {
		return err
	}

	mailserver, err := mailmgmt.CONFIG.InspectDockerMailServer(ctx, false)
	if err != nil {
		return err
	}

	opts := client.CopyToContainerOptions{
		DestinationPath: "/tmp/docker-mailserver",
		Content:         &b,
	}

	_, err = mailmgmt.CONFIG.Docker.CopyToContainer(ctx, mailserver.Container.ID, opts)
	if err != nil {
		return fmt.Errorf("failed to copy sieve script: %w", err)
	}

	return nil
}
