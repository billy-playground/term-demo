/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package track

import (
	"context"
	"io"

	"github.com/billy-playground/term-demo/progress"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
)

// todo: need a trackable interface

type target struct {
	oras.Target
	m              progress.Manager
	actionPrompt   string
	completePrompt string
}

func NewTarget(t oras.Target, actionPrompt, completePromt string) (*target, error) {
	manager, err := progress.NewManager()
	if err != nil {
		return nil, err
	}

	return &target{
		Target:         t,
		m:              manager,
		actionPrompt:   actionPrompt,
		completePrompt: completePromt,
	}, nil
}

func (t *target) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	r, err := managedReader(content, expected, t.m, t.actionPrompt, t.completePrompt)
	if err != nil {
		return err
	}
	return t.Target.Push(ctx, expected, r)
}

func (t *target) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	r, err := managedReader(content, expected, t.m, t.actionPrompt, t.completePrompt)
	if err != nil {
		return err
	}

	if rp, ok := t.Target.(registry.ReferencePusher); ok {
		err = rp.PushReference(ctx, expected, r, reference)
	} else {
		if err := t.Target.Push(ctx, expected, r); err != nil {
			return err
		}
		err = t.Target.Tag(ctx, expected, reference)
	}
	return err
}

func (t *target) Wait() {
	t.m.Wait()
}
