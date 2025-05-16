package sometext

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"inaba.kiyuri.ca/2025/convind/data"
)

type HandlerFunc func(data.DataRevision) ([]string, error)

var errNotMatch = errors.New("does not match")

// MakePrefixHandler returns a [handlerFunc] with the commandTemplate given.
// The command is run with stdin as the file.
func MakePrefixHandler(prefix string, command []string) HandlerFunc {
	return func(dr data.DataRevision) ([]string, error) {
		if !strings.HasPrefix(dr.Data().MIMEType(), prefix) {
			return nil, errNotMatch
		}
		return command, nil
	}
}

// SometextClass is a customizable class that runs arbitrary commands.
type SometextClass struct {
	name string
	// handlers is the list of handlers to attempt on each [data.DataRevision].
	// Handlers are attempted first to last; if two handlers match, the first one will be chosen.
	handlers       []HandlerFunc
	outputMIMEType string
}

func NewSometextClass(name string, handlers []HandlerFunc, mimeType string) *SometextClass {
	return &SometextClass{name, handlers, mimeType}
}

func (s *SometextClass) Name() string { return s.name }

func (s *SometextClass) AttemptInstance(dr data.DataRevision) (data.Instance, error) {
	for _, f := range s.handlers {
		command, err := f(dr)
		if err != nil {
			continue
		}
		// cachePath just has to be a function of DataRevision and the instance or command
		cachePath := filepath.Join(os.TempDir(), dr.Data().ID().String()+strconv.FormatUint(dr.RevisionID(), 10)+base64.URLEncoding.EncodeToString([]byte(fmt.Sprint(command))))
		return &commandInstance{dr, s, command, cachePath}, nil
	}
	return nil, errors.New("no matched handlers")
}

type passthroughInstance struct {
	dr data.DataRevision
}

func (i *passthroughInstance) DataRevision() data.DataRevision { return i.dr }

func (i *passthroughInstance) MIMEType() string { return i.dr.Data().MIMEType() }

func (i *passthroughInstance) NewReadCloser() (io.ReadCloser, error) {
	return i.dr.NewReadCloser()
}

type commandInstance struct {
	dr        data.DataRevision
	c         *SometextClass
	command   []string
	cachePath string
}

func (i *commandInstance) DataRevision() data.DataRevision { return i.dr }

func (i *commandInstance) MIMEType() string {
	if i.c.outputMIMEType == "PASSTHROUGH" {
		return i.dr.Data().MIMEType()
	}
	return i.c.outputMIMEType
}

func (i *commandInstance) NewReadCloser() (io.ReadCloser, error) {
	f, err := os.Open(i.cachePath)
	if err == nil {
		return f, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err = os.Create(i.cachePath)
	if err != nil {
		return nil, fmt.Errorf("create cache file: %w", err)
	}
	stdin, err := i.dr.NewReadCloser()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("NewReadCloser: %w", err)
	}
	defer stdin.Close()
	log.Printf("running %v", i.command)
	cmd := exec.Command(i.command[0], i.command[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = f
	err = cmd.Run()
	if err != nil {
		f.Close()
		return nil, err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		f.Close()
		return nil, err
	}
	// TODO: nicely handle errors while output is being written?
	return f, nil
}

type buffer struct {
	*bytes.Buffer
}

func (b *buffer) Close() error { return nil }
