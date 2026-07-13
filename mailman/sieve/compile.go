package sieve

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"

	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
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

	bannedEmailRows, err := queries.GetQuerySet(&BannedEmail{}).All()
	if err != nil {
		return nil, err
	}

	forwardedRows, err := queries.GetQuerySet(&ForwardedEmail{}).All()
	if err != nil {
		return nil, err
	}

	vacationRuleRows, err := queries.GetQuerySet(&VacationRule{}).Select("*", "For.*").All()
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

func Compile(ctx context.Context, config *SieveConfigData, dest io.Writer) error {
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

	if err := tmpl.Execute(dest, config); err != nil {
		return fmt.Errorf("failed to execute sieve template: %w", err)
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

	if err := UploadSieveToMailserver(ctx, &b); err != nil {
		return err
	}

	return nil
}
