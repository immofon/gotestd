package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dietsche/rfsnotify"
	"gopkg.in/fsnotify.v1"
)

type Info struct {
	Size    int64
	ModTime time.Time
}

type ColorWriter struct {
	io.Writer
	Color string
}

func (cw *ColorWriter) Write(p []byte) (n int, err error) {
	index := "0"
	switch cw.Color {
	case "red":
		index = "31"
	case "green":
		index = "32"
	default:
	}
	N := 0
	n, err = cw.Writer.Write([]byte("\u001b[" + index + "m"))
	if err != nil {
		return
	}

	N += n
	n, err = cw.Writer.Write(p)
	if err != nil {
		return N + n, err
	}

	N += n

	n, err = cw.Writer.Write([]byte("\u001b[0m"))
	if err != nil {
		return N + n, err
	}

	N += n

	return N, nil
}

func main() {
	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	//rfsnotify adds two new API entry points:
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	watcher.AddRecursive(pwd)

	restart := make(chan string, 16)
	restart <- ""

	infos := make(map[string]*Info)
	var (
		ctx    context.Context
		cancel context.CancelFunc = func() {}
	)
	go func() {
		for name := range restart {
			stat, err := os.Stat(name)
			if err == nil {
				info := infos[name]
				if info == nil {
					info = &Info{}
				}

				if info.Size == stat.Size() &&
					info.ModTime == stat.ModTime() {
					continue
				}

				info.Size = stat.Size()
				info.ModTime = stat.ModTime()

			}
			delete(infos, name)

			cancel()

			ctx, cancel = context.WithCancel(context.Background())
			cmd := exec.CommandContext(ctx, "go", "test", "-v", "./...")
			cmd.Stdout = &ColorWriter{os.Stdout, "green"}
			cmd.Stderr = &ColorWriter{os.Stderr, "red"}
			go cmd.Run()
		}
	}()

	for {
		select {
		case e := <-watcher.Events:
			if e.Op == fsnotify.Chmod {
				continue
			}
			if strings.HasSuffix(e.Name, ".go") {
				restart <- e.Name
			}
		}
	}
}

/*

// Event represents a single file system notification.
type Event struct {
	Name string // Relative path to the file or directory.
	Op   Op     // File operation that triggered the event.
}

// Op describes a set of file operations.
type Op uint32

// These are the generalized file operations that can trigger a notification.
const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)



*/
