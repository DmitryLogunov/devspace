package docker

import (
	"os"
	"strings"

	"context"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/docker/registry"

	"github.com/covexo/devspace/pkg/util/log"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"

	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/pkg/errors"

	"github.com/docker/docker/pkg/jsonmessage"
)

// Builder holds the necessary information to build and push docker images
type Builder struct {
	RegistryURL string
	ImageName   string
	ImageTag    string

	imageURL   string
	authConfig *types.AuthConfig
	client     client.CommonAPIClient
}

// NewBuilder creates a new docker Builder instance
func NewBuilder(registryURL, imageName, imageTag string, preferMinikube bool) (*Builder, error) {
	var cli client.CommonAPIClient
	var err error

	if preferMinikube {
		cli, err = newDockerClientFromMinikube()
	}
	if preferMinikube == false || err != nil {
		cli, err = newDockerClientFromEnvironment()

		if err != nil {
			return nil, err
		}
	}

	imageURL := imageName + ":" + imageTag
	if registryURL != "" {
		// Check if it's the official registry or not
		ref, err := reference.ParseNormalizedNamed(registryURL + "/" + imageURL)
		if err != nil {
			return nil, err
		}

		repoInfo, err := registry.ParseRepositoryInfo(ref)
		if err != nil {
			return nil, err
		}

		if repoInfo.Index.Official == false {
			imageURL = registryURL + "/" + imageURL
		}
	}

	return &Builder{
		RegistryURL: registryURL,
		ImageName:   imageName,
		ImageTag:    imageTag,
		imageURL:    imageURL,
		client:      cli,
	}, nil
}

// BuildImage builds a dockerimage with the docker cli
// contextPath is the absolute path to the context path
// dockerfilePath is the absolute path to the dockerfile WITHIN the contextPath
func (b *Builder) BuildImage(contextPath, dockerfilePath string, options *types.ImageBuildOptions) error {
	if options == nil {
		options = &types.ImageBuildOptions{}
	}

	ctx := context.Background()
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(contextPath, dockerfilePath)
	if err != nil {
		return err
	}

	excludes, err := build.ReadDockerignore(contextDir)
	if err != nil {
		return err
	}

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return errors.Errorf("Error checking context: '%s'", err)
	}

	// And canonicalize dockerfile name to a platform-independent one
	authConfigs, _ := getAllAuthConfigs()
	relDockerfile, err = archive.CanonicalTarNameForPath(relDockerfile)
	if err != nil {
		return err
	}

	excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, false)
	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.IDPair{UID: 0, GID: 0},
	})
	if err != nil {
		return err
	}

	// Setup an upload progress bar
	progressOutput := streamformatter.NewProgressOutput(log.GetInstance())
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")
	response, err := b.client.ImageBuild(ctx, body, types.ImageBuildOptions{
		Tags:        []string{b.imageURL},
		Dockerfile:  relDockerfile,
		BuildArgs:   options.BuildArgs,
		AuthConfigs: authConfigs,
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	fd, _ := term.GetFdInfo(os.Stdout)
	err = jsonmessage.DisplayJSONMessagesStream(response.Body, log.GetInstance(), fd, false, nil)
	if err != nil {
		return err
	}

	return nil
}

// Authenticate authenticates the client with a remote registry
func (b *Builder) Authenticate(user, password string, checkCredentialsStore bool) error {
	ctx := context.Background()
	authServer := getOfficialServer(ctx, b.client)
	serverAddress := b.RegistryURL

	if serverAddress == "" {
		serverAddress = authServer
	} else {
		ref, err := reference.ParseNormalizedNamed(b.imageURL)
		if err != nil {
			return err
		}

		repoInfo, err := registry.ParseRepositoryInfo(ref)
		if err != nil {
			return err
		}

		if repoInfo.Index.Official {
			serverAddress = authServer
		}
	}

	authConfig, err := getDefaultAuthConfig(b.client, checkCredentialsStore, serverAddress, serverAddress == authServer)
	if err != nil || authConfig.Username == "" || authConfig.Password == "" {
		authConfig.Username = strings.TrimSpace(user)
		authConfig.Password = strings.TrimSpace(password)
	}

	response, err := b.client.RegistryLogin(ctx, *authConfig)
	if err != nil {
		return err
	}

	if response.IdentityToken != "" {
		authConfig.Password = ""
		authConfig.IdentityToken = response.IdentityToken
	}

	b.authConfig = authConfig
	return nil
}

// PushImage pushes an image to the specified registry
func (b *Builder) PushImage() error {
	ctx := context.Background()
	ref, err := reference.ParseNormalizedNamed(b.imageURL)
	if err != nil {
		return err
	}

	encodedAuth, err := encodeAuthToBase64(*b.authConfig)
	if err != nil {
		return err
	}

	out, err := b.client.ImagePush(ctx, reference.FamiliarString(ref), types.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}

	fd, _ := term.GetFdInfo(os.Stdout)
	err = jsonmessage.DisplayJSONMessagesStream(out, log.GetInstance(), fd, false, nil)
	if err != nil {
		return err
	}

	return nil
}
