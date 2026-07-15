//go:build dockersock
// +build dockersock

package mailmgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Nigel2392/cache"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/moby/moby/api/types/container"
)

func MailServer(ctx context.Context, refresh bool) (*container.InspectResponse, error) {
	cli, err := docker.DockerErr()
	if err != nil {
		return nil, err
	}

	var (
		result  container.InspectResponse
		rawData []byte
		cacheV  any
		ok      bool
	)

	if refresh {
		goto notCached
	}

	cacheV, err = cache.Get(ctx, _MAILSERVER_CONTAINER_INSPECT_CACHE_KEY)
	if err != nil && !errors.Is(err, cache.ErrItemNotFound) {
		return nil, err
	}

	rawData, ok = cacheV.([]byte)
	if !ok || len(rawData) == 0 {
		goto notCached
	}

	err = json.Unmarshal(rawData, &result)
	if err != nil {
		return nil, err
	}

notCached:
	containerName := django.ConfigGet(
		django.Global.Settings,
		MAILSERVER_CONTAINER_NAME,
		"mailserver",
	)

	res, err := cli.ContainerInspect(
		ctx, containerName,
		client.ContainerInspectOptions{},
	)
	if err != nil {
		return nil, ErrDocker.WithCause(fmt.Errorf(
			"could not retrieve container %q, are you sure it is running? %w",
			containerName, err,
		))
	}

	cache.Set(
		ctx, _MAILSERVER_CONTAINER_INSPECT_CACHE_KEY, []byte(res.Raw), time.Minute*5,
	)

	return &res.Container, nil
}
