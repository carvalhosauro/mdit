package doc

import (
	"errors"
	"os"
	"time"
)

// ErrExternalChange is returned by Save when the file's mtime on disk no
// longer matches the mtime recorded at Load or at the last successful Save.
var ErrExternalChange = errors.New("file changed on disk")

// errNoPath is returned by Save/SaveForce when the document has no
// associated file path (created via NewFromString and never saved).
var errNoPath = errors.New("doc: no path set; load a file or provide a path before saving")

// Load reads path into a new Document. A nonexistent path is not an error:
// it yields an empty document (one empty line) with Path set to path; the
// file is created on the first Save.
func Load(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Document{
				lines:   []string{""},
				path:    path,
				hasPath: true,
				now:     time.Now,
			}, nil
		}
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &Document{
		lines:   splitLines(string(data)),
		path:    path,
		hasPath: true,
		modTime: info.ModTime(),
		now:     time.Now,
	}, nil
}

// Save writes the document to Path if the file on disk has not changed
// since Load or the last successful Save (compared by mtime). If it has,
// Save returns ErrExternalChange without writing anything. Use SaveForce to
// overwrite unconditionally.
func (d *Document) Save() error {
	if !d.hasPath || d.path == "" {
		return errNoPath
	}
	info, err := os.Stat(d.path)
	if err == nil {
		if !info.ModTime().Equal(d.modTime) {
			return ErrExternalChange
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return d.writeFile()
}

// SaveForce writes the document to Path unconditionally, ignoring any
// external modification.
func (d *Document) SaveForce() error {
	if !d.hasPath || d.path == "" {
		return errNoPath
	}
	return d.writeFile()
}

// writeFile performs the actual write and re-stats the file so d.modTime
// reflects exactly what's on disk (avoiding filesystem timestamp-resolution
// races), then marks the document clean at the current Version.
func (d *Document) writeFile() error {
	if err := os.WriteFile(d.path, []byte(d.Content()), 0o644); err != nil {
		return err
	}
	info, err := os.Stat(d.path)
	if err != nil {
		return err
	}
	d.modTime = info.ModTime()
	d.savedVersion = d.version
	return nil
}
