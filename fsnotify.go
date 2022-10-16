// Package fsnotify provides a cross-platform interface for file system
// notifications.
//
// Currently supported systems:
//
//    Linux 2.6.32+    via inotify
//    BSD, macOS       via kqueue
//    Windows          via ReadDirectoryChangesW
//    illumos          via FEN
package fsnotify

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// Event represents a file system notification.
type Event struct {
	// Path to the file or directory.
	//
	// Paths are relative to the input; for example with Add("dir") the Name
	// will be set to "dir/file" if you create that file, but if you use
	// Add("/path/to/dir") it will be "/path/to/dir/file".
	Name string

	// File operation that triggered the event.
	//
	// This is a bitmask and some systems may send multiple operations at once.
	// Use the Event.Has() method instead of comparing with ==.
	Op Op
}

// Op describes a set of file operations.
type Op uint32

// The operations fsnotify can trigger; see the documentation on [Watcher] for a
// full description, and check them with [Event.Has].
const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

// Common errors that can be reported.
var (
	ErrNonExistentWatch     = errors.New("fsnotify: can't remove non-existent watcher")
	ErrEventOverflow        = errors.New("fsnotify: queue or buffer overflow")
	ErrClosed               = errors.New("fsnotify: watcher already closed")
	ErrNotDirectory         = errors.New("fsnotify: not a directory")
	ErrRecursionUnsupported = errors.New("fsnotify: recursion not supported")
)

func (o Op) String() string {
	var b strings.Builder
	if o.Has(Create) {
		b.WriteString("|CREATE")
	}
	if o.Has(Remove) {
		b.WriteString("|REMOVE")
	}
	if o.Has(Write) {
		b.WriteString("|WRITE")
	}
	if o.Has(Rename) {
		b.WriteString("|RENAME")
	}
	if o.Has(Chmod) {
		b.WriteString("|CHMOD")
	}
	if b.Len() == 0 {
		return "[no events]"
	}
	return b.String()[1:]
}

// Has reports if this operation has the given operation.
func (o Op) Has(h Op) bool { return o&h == h }

// Has reports if this event has the given operation.
func (e Event) Has(op Op) bool { return e.Op.Has(op) }

// String returns a string representation of the event with their path.
func (e Event) String() string {
	return fmt.Sprintf("%-13s %q", e.Op.String(), e.Name)
}

// findDirs finds all directories under path (return value *includes* path as
// the first entry).
//
// A symlink for a directory is not considered a directory.
func findDirs(path string) ([]string, error) {
	dirs := make([]string, 0, 8)
	err := filepath.WalkDir(path, func(root string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if root == path && !d.IsDir() {
			return fmt.Errorf("%q: %w", path, ErrNotDirectory)
		}
		if d.IsDir() {
			dirs = append(dirs, root)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dirs, nil
}

// Check if this path is recursive (ends with "/..."), and return the path with
// the /... stripped.
func recursivePath(path string) (string, bool) {
	if filepath.Base(path) == "..." {
		return filepath.Dir(path), true
	}
	return path, false
}
