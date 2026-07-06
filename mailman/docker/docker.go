package docker

import (
	"errors"

	django "github.com/Nigel2392/go-django/src"
	"github.com/moby/moby/client"
)

const APPVAR_DOCKER_CLIENT = "docker.APPVAR_DOCKER_CLIENT" // func() (*client.Client, error) || *client.Client

var D *client.Client

// This function can retrieve and/or instantiate a docker client.
// The docker client will be stored in a global variable.
func DockerErr() (*client.Client, error) {
	if D == nil {
		ini, ok := django.ConfigGetOK[any](
			django.Global.Settings,
			APPVAR_DOCKER_CLIENT,
		)
		if !ok {
			return nil, errors.New(
				"docker client not in settings, configure 'docker.APPVAR_DOCKER_CLIENT'",
			)
		}

		var (
			f   *client.Client
			err error
		)
		switch t := ini.(type) {
		case func() (*client.Client, error):
			f, err = t()
		case *client.Client:
			f = t
		}

		if err != nil {
			return nil, err
		}

		D = f
	}

	return D, nil
}

// This function can retrieve and/or instantiate a docker client.
// The docker client will be stored in a global variable.
// If an error occurs during initialization, this function panics.
func Docker() *client.Client {
	r, err := DockerErr()
	if err != nil {
		panic(err)
	}
	return r
}
