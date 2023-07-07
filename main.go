package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/term"

	"github.com/billy-playground/term-demo/track"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	credentials "github.com/oras-project/oras-credentials-go"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

const (
	helpBlob = `usage: blob <ref> [output-path]`
	helpCp   = `usage: cp <ref-from> <ref-to>`
)

func main() {
	args := os.Args[1:]
	if len(args) != 0 {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		switch args[0] {
		case "blob":
			args = args[1:]
			args = append(args, "")
			if err := fetchBlob(ctx, args[0], args[1]); err != nil {
				fmt.Fprintln(os.Stderr, "failed to fetch blob:", err)
				os.Exit(1)
			}
			return
		case "cp":
			args = args[1:]
			if len(args) != 2 {
				fmt.Println(helpCp)
			}
			err, ref := copy(ctx, args[0], args[1])
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to copy:", err)
				os.Exit(1)
			}
			fmt.Println("Copied", ref)
			return
		}

	}
	fmt.Println(helpBlob)
	fmt.Println(helpCp)
	os.Exit(1)

}

func PlainHTTP(registry string) bool {
	return strings.HasPrefix(registry, "localhost")
}

func fetchBlob(ctx context.Context, ref, outputPath string) (fetchErr error) {
	var target oras.ReadOnlyTarget
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return err
	}
	repo.PlainHTTP = PlainHTTP(repo.Reference.Registry)
	target = repo.Blobs()
	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return err
	}
	auth.DefaultClient.Credential = credentials.Credential(store)

	var desc ocispec.Descriptor
	if outputPath == "" {
		// fetch blob descriptor only
		_, err = oras.Resolve(ctx, target, repo.Reference.Reference, oras.DefaultResolveOptions)
		if err != nil {
			return err
		}
	} else {
		// fetch blob content
		var rc io.ReadCloser
		desc, rc, err = oras.Fetch(ctx, target, repo.Reference.Reference, oras.DefaultFetchOptions)
		if err != nil {
			return err
		}
		defer rc.Close()
		vr := content.NewVerifyReader(rc, desc)

		// outputs blob content if "--output -" is used
		if outputPath == "-" {
			if _, err := io.Copy(os.Stdout, vr); err != nil {
				return err
			}
			return vr.Verify()
		}

		// save blob content into the local file if the output path is provided
		var r io.Reader = vr
		var waitFunc func()
		if term.IsTerminal(int(os.Stdout.Fd())) {
			trackedReader, err := track.NewReader(r, desc, "Downloading", "Downloaded")
			if err != nil {
				return err
			}
			r = trackedReader
			waitFunc = func() {
				trackedReader.Wait()
			}
			defer waitFunc()

		} else {
			fmt.Println("not a terminal")
		}

		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer func() {
			if err := file.Close(); fetchErr == nil {
				fetchErr = err
			}
		}()

		if _, err := io.Copy(file, r); err != nil {
			return err
		}
		if err := vr.Verify(); err != nil {
			return err
		}

		waitFunc()
		fmt.Println("Blob saved to", outputPath)
	}
	return nil
}
