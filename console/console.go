// from github.com/containerd/console/console_unix.go
// TODO: this file is mosted unix-specific, needing windows

package console

import (
	"fmt"
	"os"

	"github.com/containerd/console"
	"github.com/morikuni/aec"
)

// Console is a wrapper around containerd's console.Console and ANSI escape
// codes.
type Console struct {
	console.Console
}

func (c *Console) Size() (width, height int) {
	width = 80
	height = 10
	size, err := c.Console.Size()
	if err == nil && size.Height > 0 && size.Width > 0 {
		width = int(size.Width)
		height = int(size.Height)
	}
	return
}

func GetConsole(f *os.File) (*Console, error) {
	c, err := console.ConsoleFromFile(f)
	if err != nil {
		return nil, err
	}
	return &Console{c}, nil
}

func (c *Console) Save() {
	fmt.Fprint(c, aec.Hide)
	// cannot use aec.Save since DEC has better support than SCO
	fmt.Fprint(c, "\0337")
}

func (c *Console) NewRow() {
	// cannot use aec.Restore since DEC has better support than SCO
	fmt.Fprint(c, "\0338")
	fmt.Fprint(c, aec.Down(1))
	fmt.Fprint(c, aec.Column(0))
	fmt.Fprint(c, "\0337")
}

func (c *Console) OutputTo(upCnt uint, str string) {
	// fmt.Fprint(c, aec.Restore)
	fmt.Fprint(c, "\0338")
	fmt.Fprint(c, aec.Up(upCnt))
	fmt.Fprint(c, aec.Column(0))
	fmt.Fprint(c, str+" ")
	fmt.Fprint(c, aec.EraseLine(aec.EraseModes.Tail))
}

func (c *Console) Restore() {
	fmt.Fprint(c, "\0338")
	fmt.Fprint(c, aec.Column(0))
	fmt.Fprint(c, aec.EraseLine(aec.EraseModes.All))
	fmt.Fprint(c, aec.Show)
}
