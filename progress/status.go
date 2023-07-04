package progress

import (
	"fmt"

	"github.com/dustin/go-humanize"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var spinner = []rune("⠋⠋⠙⠙⠹⠹⠸⠸⠼⠼⠴⠴⠦⠦⠧⠧⠇⠇⠏⠏")
var spinnerLen = len(spinner)

var spinnerPos = 0

// status is a progress status
type status struct {
	prompt     string
	descriptor ocispec.Descriptor
	offset     uint64
}

func NewStatus(prompt string, descriptor ocispec.Descriptor, offset uint64) *status {
	return &status{
		prompt:     prompt,
		descriptor: descriptor,
		offset:     offset,
	}
}

// String returns a viewable TTY string of the status.
func (s *status) String(width int) string {
	if s == nil {
		return "loading..."
	}

	current := s.offset
	total := uint64(s.descriptor.Size)
	d := s.descriptor.Digest.Encoded()[:12]
	percent := float64(s.offset) / float64(total)

	name := s.descriptor.Annotations["org.opencontainers.image.title"]
	if name == "" {
		name = s.descriptor.MediaType
	}
	left := fmt.Sprintf("%s %s %s", s.prompt, d, name)
	right := fmt.Sprintf(" %s/%s %.2f%%", humanize.Bytes(current), humanize.Bytes(total), percent*100)
	if len(left)+len(right) > width {
		right = fmt.Sprintf(" %.2f%%", percent*100)
	}

	if s.offset != uint64(s.descriptor.Size) {
		spinnerPos = (spinnerPos + 2) % spinnerLen
		right = fmt.Sprintf("%s %c", right, spinner[spinnerPos])
	}
	return fmt.Sprintf("%-*s%s", width-len(right)-1, left, right)
}
