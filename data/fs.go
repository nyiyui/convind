package data

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type FSDataStore struct {
	prefix string
}

var _ DataStore = (*FSDataStore)(nil)

// NewFSDataStoreFromSubdirectory returns a new FSDataStore using [os.DirFS].
func NewFSDataStoreFromSubdirectory(directory string) *FSDataStore {
	return &FSDataStore{directory}
}

func (f *FSDataStore) GetDataByID(id ID) (Data, error) {
	_, err := os.Stat(filepath.Join(f.prefix, id.String()))
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(f.prefix, id.String(), ".datatype"))
	if errors.Is(err, os.ErrNotExist) {
		raw = []byte("application/octet-stream")
	} else if err != nil {
		return nil, fmt.Errorf("reading .datatype: %w", err)
	}
	return &FSData{f.prefix, id, string(raw)}, nil
}

func (f *FSDataStore) New(mimeType string) (Data, error) {
	id := GenerateRandomID()
	err := os.Mkdir(filepath.Join(f.prefix, id.String()), 0700)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(filepath.Join(f.prefix, id.String(), ".datatype"), []byte(mimeType), 0600)
	if err != nil {
		return nil, err
	}
	return &FSData{f.prefix, id, mimeType}, nil
}

func (f *FSDataStore) AllIDs() ([]ID, error) {
	entries, err := os.ReadDir(f.prefix)
	if err != nil {
		return nil, err
	}
	ids := make([]ID, len(entries))
	for i, entry := range entries {
		if entry.Name()[0] == '.' {
			continue
		}
		ids[i], err = ParseID(entry.Name())
		if err != nil {
			return nil, err
		}
	}
	return ids, nil
}

type FSData struct {
	prefix   string
	id       ID
	mimeType string
}

var _ Data = (*FSData)(nil)

func (f *FSData) ID() ID {
	return f.id
}

func (f *FSData) Revisions() ([]DataRevision, error) {
	entries, err := os.ReadDir(filepath.Join(f.prefix, f.id.String()))
	if err != nil {
		return nil, err
	}
	revisions := make([]DataRevision, 0, len(entries))
	for _, entry := range entries {
		if entry.Name()[0] != '.' {
			revisionID, err := strconv.ParseUint(entry.Name(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse revision id of %s: %w", entry.Name(), err)
			}
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("stat %s", entry.Name())
			}
			revisions = append(revisions, &FSRevision{f.prefix, f.id, info, revisionID, f.mimeType})
		}
	}
	return revisions, nil
}

func (f *FSData) NewRevision(r io.Reader) (DataRevision, error) {
	revisionID := GenerateRandomID().Random
	file, err := os.Create(filepath.Join(f.prefix, f.id.String(), strconv.FormatUint(revisionID, 10)))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	_, err = io.Copy(file, r)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return &FSRevision{f.prefix, f.id, info, revisionID, f.mimeType}, nil
}

func (f *FSData) MIMEType() string { return strings.TrimSpace(f.mimeType) }

func (f *FSData) MarshalJSON() ([]byte, error) {
	return MarshalData(f)
}

type FSRevision struct {
	prefix     string
	id         ID
	info       fs.FileInfo
	revisionID uint64
	mimeType   string
}

func (f *FSRevision) Data() Data {
	return &FSData{f.prefix, f.id, f.mimeType}
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
	return os.Open(filepath.Join(f.prefix, f.id.String(), strconv.FormatUint(f.revisionID, 10)))
}
