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
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/billy-playground/term-demo/progress"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type reader struct {
	base           io.Reader
	offset         atomic.Uint64
	actionPrompt   string
	completePrompt string
	descriptor     ocispec.Descriptor
	mu             sync.Mutex
	m              progress.Manager
	terminated     atomic.Bool
	progress       progress.Progress
}

func padPrompt(prompt1, prompt2 string) (string, string) {
	len1 := len(prompt1)
	len2 := len(prompt2)
	if len1 > len2 {
		return prompt1, fmt.Sprintf("%-*s", len1, prompt2)
	}
	return fmt.Sprintf("%-*s", len2, prompt1), prompt2
}

func NewReader(r io.Reader, descriptor ocispec.Descriptor, actionPrompt, completePromt string) (*reader, error) {
	manager, err := progress.NewManager()
	if err != nil {
		return nil, err
	}
	return managedReader(r, descriptor, manager, actionPrompt, completePromt)
}

func managedReader(r io.Reader, descriptor ocispec.Descriptor, manager progress.Manager, actionPrompt, completePromt string) (*reader, error) {
	actionPrompt, completePromt = padPrompt(actionPrompt, completePromt)
	return &reader{
		base:           r,
		descriptor:     descriptor,
		actionPrompt:   actionPrompt,
		completePrompt: completePromt,
		m:              manager,
		progress:       manager.Add(),
	}, nil
}

func (r *reader) Wait() {
	r.m.Wait()
}

func (r *reader) Read(p []byte) (int, error) {
	n, err := r.base.Read(p)
	if err != nil && err != io.EOF {
		// unexpected error whil reading
		if !r.terminated.Swap(true) {
			close(r.progress)
		}
		return n, err
	}

	offset := r.offset.Add(uint64(n))
	if err == io.EOF {
		if offset == uint64(r.descriptor.Size) {
			if !r.terminated.Swap(true) {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.progress <- progress.NewStatus(r.completePrompt, r.descriptor, offset)
				close(r.progress)
			}
			return n, err
		}
		return n, io.ErrUnexpectedEOF
	}

	if r.mu.TryLock() {
		defer r.mu.Unlock()
		if len(r.progress) < progress.BUFFER_SIZE {
			// intermediate progress might be ignored if buffer is full
			r.progress <- progress.NewStatus(r.actionPrompt, r.descriptor, offset)
		}
	}
	return n, err
}
