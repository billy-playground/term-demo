package progress

import (
	"os"
	"sync"
	"time"

	"github.com/billy-playground/term-demo/console"
)

const BUFFER_SIZE = 20

// Progress is print message channel
type Progress chan<- *status

// Manager is progress view master
type Manager interface {
	Add() Progress
	Wait()
}

const (
	bufFlushDuration = 100 * time.Millisecond
)

type manager struct {
	statuses []*status

	// todo: use separate routine to update size
	// width, height int
	done       chan struct{}
	renderTick *time.Ticker
	c          *console.Console
	updating   sync.WaitGroup
	sync.WaitGroup
	mu    sync.Mutex
	close sync.Once
}

// NewManager initialized a new progress manager
func NewManager() (Manager, error) {
	var m manager
	var err error
	// f, err := os.OpenFile("/dev/pts/3", os.O_RDWR, 0)
	// if err != nil {
	// 	log.Fatalf("%+v", err)
	// 	return nil, err
	// }
	// f.Write([]byte("manager created"))
	// m.c, err = console.GetConsole(f)

	m.c, err = console.GetConsole(os.Stdout)
	if err != nil {
		return nil, err
	}
	m.done = make(chan struct{})
	m.renderTick = time.NewTicker(bufFlushDuration)
	m.start()
	return &m, nil
}

func (m *manager) start() {
	m.renderTick.Reset(bufFlushDuration)
	m.c.Save()
	go func() {
		for {
			m.render()
			select {
			case <-m.done:
				return
			case <-m.renderTick.C:
			}
		}
	}()
}

func (m *manager) render() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// todo: update size in another routine
	width, height := m.c.Size()
	len := len(m.statuses)
	offset := 0
	if len > height {
		// skip statuses that cannot be rendered
		offset = len - height
	}

	for ; offset < len; offset++ {
		m.c.OutputTo(uint(len-offset), m.statuses[offset].String(width))
	}
}

func (m *manager) Add() Progress {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := len(m.statuses)

	var s *status
	m.statuses = append(m.statuses, s)
	progress := make(chan *status, BUFFER_SIZE)
	m.updating.Add(1)
	go func() {
		defer m.updating.Done()
		for s := range progress {
			m.statuses[id] = s
		}
	}()

	// add a new line to the end cursor
	m.c.NewRow()
	return progress
}

func (m *manager) Wait() {
	m.close.Do(func() {
		// 1. stop ticking
		m.renderTick.Stop()
		// 2. wait for all model update done
		m.updating.Wait()
		// 3. render last model
		m.render()
		// 4. restore cursor, mark done
		defer m.c.Restore()
		close(m.done)
	})
}
