//go:build dockersock
// +build dockersock

package sieve

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/docker"
	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
	"github.com/moby/moby/client"
)

func UploadSieveToMailserver(ctx context.Context, b *bytes.Buffer) error {
	var (
		dest        = new(bytes.Buffer)
		tarWriter   = tar.NewWriter(dest)
		scriptBytes = b.Bytes()
	)
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

	mailserver, err := mailmgmt.MailServer(ctx, true)
	if err != nil {
		return err
	}

	opts := client.CopyToContainerOptions{
		DestinationPath: SIEVE_DIRECTORY,
		Content:         b,
	}

	_, err = docker.Docker().CopyToContainer(ctx, mailserver.ID, opts)
	if err != nil {
		return fmt.Errorf("failed to copy sieve script: %w", err)
	}

	return nil
}
