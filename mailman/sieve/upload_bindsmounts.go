//go:build !dockersock
// +build !dockersock

package sieve

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func UploadSieveToMailserver(ctx context.Context, b *bytes.Buffer) error {
	if err := os.MkdirAll(SIEVE_DIRECTORY, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("'%s' directory creation failed: %v", SIEVE_DIRECTORY, err)
	}

	var path = filepath.Join(SIEVE_DIRECTORY, "before.dovecot.sieve")
	var f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open '%s': %w", path, err)
	}
	defer f.Close()

	_, err = io.Copy(f, b)
	if err != nil {
		return fmt.Errorf("failed to write to '%s': %w", path, err)
	}

	return nil
}
