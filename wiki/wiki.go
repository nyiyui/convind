package wiki

import (
	"bytes"
	"io"
	"strconv"

	"github.com/yuin/goldmark"
	"inaba.kiyuri.ca/2025/convind/data"
)

type Page struct {
	Data data.Data
}

// TODO: NewPage func that checks MIME type

func (p *Page) URL() string {
	return "convind://" + p.Data.ID().String()
}

func (p *Page) Revisions() ([]*PageRevision, error) {
	revisions, err := p.Data.Revisions()
	if err != nil {
		return nil, err
	}
	result := make([]*PageRevision, len(revisions))
	for i, revision := range revisions {
		result[i] = &PageRevision{revision}
	}
	return result, nil
}

// LatestRevision returns the latest revision if available, and nil is there are no revisions at all.
func (p *Page) LatestRevision() (*PageRevision, error) {
	revisions, err := p.Data.Revisions()
	if err != nil {
		return nil, err
	}
	var latestRevision data.DataRevision
	for _, revision := range revisions {
		if latestRevision == nil || revision.CreationTime().After(latestRevision.CreationTime()) {
			latestRevision = revision
		}
	}
	if latestRevision == nil {
		return nil, nil
	}
	return &PageRevision{latestRevision}, nil
}

type PageRevision struct {
	DataRevision data.DataRevision
}

func (p *PageRevision) URL() string {
	return "convind://" + p.DataRevision.Data().ID().String() + "?revision=" + strconv.FormatUint(p.DataRevision.RevisionID(), 10)
}

func (p *PageRevision) View() (string, error) {
	rc, err := p.DataRevision.NewReadCloser()
	if err != nil {
		return "", err
	}
	source, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	md := goldmark.New()
	buf := new(bytes.Buffer)
	err = md.Convert(source, buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
