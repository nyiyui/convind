package data

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type FSDataStore struct {
	fs fs.FS
}

var _ DataStore = (*FSDataStore)(nil)

// NewFSDataStoreFromSubdirectory returns a new FSDataStore using [os.DirFS].
func NewFSDataStoreFromSubdirectory(directory string) *FSDataStore {
	return &FSDataStore{os.DirFS(directory)}
}

func (f *FSDataStore) GetDataByID(id ID) (Data, error) {
	_, err := fs.Stat(f.fs, id.String())
	if err != nil {
		return nil, err
	}
	raw, err := fs.ReadFile(f.fs, filepath.Join(id.String(), ".datatype"))
	if errors.Is(err, os.ErrNotExist) {
		raw = []byte("application/octet-stream")
	} else if err != nil {
		return nil, fmt.Errorf("reading .datatype: %w", err)
	}
	return &FSData{f.fs, id, string(raw)}, nil
}

type FSData struct {
	fs       fs.FS
	id       ID
	mimeType string
}

var _ Data = (*FSData)(nil)

func (f *FSData) ID() ID {
	return f.id
}

func (f *FSData) Revisions() ([]DataRevision, error) {
	entries, err := fs.ReadDir(f.fs, f.id.String())
	if err != nil {
		return nil, err
	}
	revisions := make([]DataRevision, len(entries))
	for i, entry := range entries {
		if entry.Name()[0] != '.' {
			revisionID, err := strconv.ParseUint(entry.Name(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse revision id of %s: %w", entry.Name(), err)
			}
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("stat %s", entry.Name())
			}
			revisions[i] = &FSRevision{f.fs, f.id, info, revisionID, f.mimeType}
		}
	}
	return revisions, nil
}

func (f *FSData) MIMEType() string { return f.mimeType }

type FSRevision struct {
	fs         fs.FS
	id         ID
	info       fs.FileInfo
	revisionID uint64
	mimeType   string
}

func (f *FSRevision) Data() Data {
	return &FSData{f.fs, f.id, f.mimeType}
}

// RevisionID is a unique number representing this revision.
// This number can be a random number, and is not necessarily incremental.
func (f *FSRevision) RevisionID() uint64 {
	return f.revisionID
}

// CreationTime returns the time this revision was created.
func (f *FSRevision) CreationTime() time.Time {
	return f.info.ModTime()
}

func (f *FSRevision) NewReadCloser() (io.ReadCloser, error) {
	return f.fs.Open(filepath.Join(f.id.String(), strconv.FormatUint(f.revisionID, 10)))
}
