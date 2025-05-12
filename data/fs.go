package data

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strconv"
	"time"
)

type FSDataStore struct {
	fs fs.FS
}

var _ DataStore = (*FSDataStore)(nil)

func (f *FSDataStore) GetDataByID(id ID) (Data, error) {
	_, err := fs.Stat(f.fs, id.String())
	if err != nil {
		return nil, err
	}
	return &FSData{f.fs, id}, nil
}

type FSData struct {
	fs fs.FS
	id ID
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
	revisions := make([]DataRevision, 0, len(entries))
	for i, entry := range entries {
		revisionID, err := strconv.ParseUint(entry.Name(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse revision id of %s: %w", entry.Name(), err)
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat %s", entry.Name())
		}
		revisions[i] = &FSRevision{f.fs, f.id, info, revisionID}
	}
	return revisions, nil
}

type FSRevision struct {
	fs         fs.FS
	id         ID
	info       fs.FileInfo
	revisionID uint64
}

func (f *FSRevision) Data() Data {
	return &FSData{f.fs, f.id}
}

// RevisionID is a unique number representing this revision.
// This number can be a random number, and is not necessarily incremental.
func (f *FSRevision) RevisionID() uint64 {
	return f.revisionID
}

// Datatype returns the MIME type of this revision.
func (f *FSRevision) Datatype() string {
	return "application/octet-stream"
}

// CreationTime returns the time this revision was created.
func (f *FSRevision) CreationTime() time.Time {
	return f.info.ModTime()
}

func (f *FSRevision) NewReadCloser() (io.ReadCloser, error) {
	return f.fs.Open(filepath.Join(f.id.String(), strconv.FormatUint(f.revisionID, 10)))
}
