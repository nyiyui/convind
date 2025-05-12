package data

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"
)

type ID struct {
	Epoch  int64 // seconds after 1970-Jan-01 UTC
	Random uint64
}

func GenerateRandomID() ID {
	maxUint64 := big.NewInt(2)
	maxUint64 = maxUint64.Exp(maxUint64, big.NewInt(64), nil)
	n, err := rand.Int(rand.Reader, maxUint64)
	if err != nil {
		panic(err)
	}
	return ID{time.Now().Unix(), n.Uint64()}
}

const idPrefix = "convind_id_"

func (i ID) asBuffer() *bytes.Buffer {
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.URLEncoding, buf)
	encoder.Write([]byte(idPrefix))
	b := make([]byte, 64/8)
	binary.LittleEndian.PutUint64(b, uint64(i.Epoch))
	_, err := encoder.Write(b)
	if err != nil {
		panic(err)
	}
	binary.LittleEndian.PutUint64(b, i.Random)
	_, err = encoder.Write(b)
	if err != nil {
		panic(err)
	}
	return buf
}

func (i ID) String() string {
	return i.asBuffer().String()
}

func (i ID) MarshalText() ([]byte, error) {
	return i.asBuffer().Bytes(), nil
}

func ParseID(s string) (ID, error) {
	i := new(ID)
	err := i.UnmarshalText([]byte(s))
	if err != nil {
		return ID{}, err
	}
	return *i, nil
}

func (i *ID) UnmarshalText(rawText []byte) (err error) {
	decoder := base64.NewDecoder(base64.URLEncoding, bytes.NewBuffer(rawText))
	text, err := io.ReadAll(decoder)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}

	if len(text) < len(idPrefix)+64/8+64/8 {
		return errors.New("too short")
	}
	if !bytes.Equal(text[:len(idPrefix)], []byte(idPrefix)) {
		return errors.New("prefix does not match expected " + idPrefix)
	}
	epoch := binary.LittleEndian.Uint64(text[len(idPrefix) : len(idPrefix)+64/8])
	if epoch > (1<<63 - 1) {
		return fmt.Errorf("epoch %d too large", epoch)
	}
	i.Epoch = int64(epoch)
	i.Random = binary.LittleEndian.Uint64(text[len(idPrefix)+64/8 : len(idPrefix)+64/8+64/8])
	return nil
}

type DataStore interface {
	GetDataByID(ID) (Data, error)
	New(mimeType string) (Data, error)
	AllIDs() ([]ID, error)
}

type Data interface {
	ID() ID
	// Revisions returns all (known) revisions, sorted newest to oldest.
	Revisions() ([]DataRevision, error)
	// NewRevision creates a new revision and returns said revision.
	NewRevision(r io.Reader) (DataRevision, error)
	// MIMEType returns the MIME type of this revision.
	MIMEType() string
	// MarshalJSON implements [json.Marshaler].
	MarshalJSON() ([]byte, error)
}

// DataRevision is a handle to a revision of data.
// All revisions are immutable, and the contents must not change.
type DataRevision interface {
	Data() Data
	// RevisionID is a unique number representing this revision.
	// This number can be a random number, and is not necessarily incremental.
	RevisionID() uint64
	// CreationTime returns the time this revision was created.
	CreationTime() time.Time
	// NewReadCloser returns an [io.ReadCloser] of this revision.
	NewReadCloser() (io.ReadCloser, error)
}

type Class interface {
	// AttemptInstance returns an instance for the given [DataRevision], if applicable.
	// If not, an error is returned.
	AttemptInstance(dr DataRevision) (Instance, error)
}

type Instance interface {
	DataRevision() DataRevision
	// NewReadCloser returns an [io.ReadCloser] of this instance.
	NewReadCloser() (io.ReadCloser, error)
}

type dataJSON struct {
	ID        ID
	Revisions []dataRevisionJSON
	MIMEType  string
}

func MarshalData(d Data) ([]byte, error) {
	obj, err := dataToJSON(d)
	if err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

func dataToJSON(d Data) (dataJSON, error) {
	revisions, err := d.Revisions()
	if err != nil {
		return dataJSON{}, err
	}
	revisions2 := make([]dataRevisionJSON, len(revisions))
	for i, revision := range revisions {
		if revision == nil {
			panic('a')
		}
		revisions2[i] = dataRevisionToJSON(revision)
	}
	return dataJSON{d.ID(), revisions2, d.MIMEType()}, nil
}

type dataRevisionJSON struct {
	RevisionID   uint64
	CreationTime time.Time
}

func dataRevisionToJSON(dr DataRevision) dataRevisionJSON {
	return dataRevisionJSON{dr.RevisionID(), dr.CreationTime()}
}
