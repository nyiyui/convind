package sometext

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"inaba.kiyuri.ca/2025/convind/data"
)

type handlerFunc func(data.DataRevision) ([]string, error)

var errNotMatch = errors.New("does not match")

// makePrefixHandler returns a [handlerFunc] with the commandTemplate given.
// The command is run with stdin as the file.
func makePrefixHandler(prefix string, command []string) handlerFunc {
	return func(dr data.DataRevision) ([]string, error) {
		if !strings.HasPrefix(dr.Data().MIMEType(), prefix) {
			return nil, errNotMatch
		}
		return command, nil
	}
}

type SometextClass struct {
	// handlers is the list of handlers to attempt on each [data.DataRevision].
	// Handlers are attempted first to last; if two handlers match, the first one will be chosen.
	handlers []handlerFunc
}

func (s *SometextClass) AttemptInstance(dr data.DataRevision) (data.Instance, error) {
	mimeType := dr.Data().MIMEType()
	if strings.HasPrefix(mimeType, "text/") {
		return &passthroughInstance{dr}, nil
	}
	for _, f := range s.handlers {
		command, err := f(dr)
		if err != nil {
			continue
		}
		// cachePath just has to be a function of DataRevision and the instance or command
		cachePath := filepath.Join(os.TempDir(), dr.Data().ID().String()+strconv.FormatUint(dr.RevisionID(), 10)+base64.URLEncoding.EncodeToString([]byte(fmt.Sprint(command))))
		return &commandInstance{dr, command, cachePath}, nil
	}
	return nil, errors.New("no matched handlers")
}

type passthroughInstance struct {
	dr data.DataRevision
}

func (i *passthroughInstance) DataRevision() data.DataRevision { return i.dr }

func (i *passthroughInstance) NewReadCloser() (io.ReadCloser, error) {
	return i.dr.NewReadCloser()
}

type commandInstance struct {
	dr        data.DataRevision
	command   []string
	cachePath string
}

func (i *commandInstance) DataRevision() data.DataRevision { return i.dr }

func (i *commandInstance) NewReadCloser() (io.ReadCloser, error) {
	f, err := os.Open(i.cachePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return f, nil
	} else if err != nil {
		return nil, err
	}
	stdin, err := i.dr.NewReadCloser()
	if err != nil {
		return nil, fmt.Errorf("NewReadCloser: %w", err)
	}
	cmd := exec.Command(i.command[0], i.command[1:]...)
	buf := new(bytes.Buffer)
	cmd.Stdin = stdin
	cmd.Stdout = buf
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	// TODO: nicely handle errors while output is being written?
	return &buffer{buf}, nil
}

type buffer struct {
	*bytes.Buffer
}

func (b *buffer) Close() error { return nil }
