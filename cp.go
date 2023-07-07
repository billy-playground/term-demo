package main

import (
	"context"
	"fmt"
	"os"

	"github.com/billy-playground/term-demo/track"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	credentials "github.com/oras-project/oras-credentials-go"
	"golang.org/x/term"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

func copy(ctx context.Context, from, to string) (error, string) {
	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return err, ""
	}
	auth.DefaultClient.Credential = credentials.Credential(store)

	src, err := remote.NewRepository(from)
	if err != nil {
		return err, ""
	}
	src.PlainHTTP = PlainHTTP(src.Reference.Registry)

	var dest oras.Target
	repo, err := remote.NewRepository(to)
	if err != nil {
		return err, ""
	}
	repo.PlainHTTP = PlainHTTP(repo.Reference.Registry)
	dest = repo

	var desc ocispec.Descriptor
	if term.IsTerminal(int(os.Stdout.Fd())) {
		tracked, err := track.NewTarget(dest, "Copying", "Copied")
		if err != nil {
			return err, ""
		}
		defer tracked.Wait()
		dest = tracked
	} else {
		fmt.Println("not a terminal, still copying")
	}
	desc, err = oras.Copy(ctx, src, from, dest, to, oras.DefaultCopyOptions)
	repo.Reference.Reference = desc.Digest.String()
	return err, repo.Reference.String()
}
