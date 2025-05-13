package wiki

import (
	"bytes"
	"encoding/json"
	"io"
	"slices"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"inaba.kiyuri.ca/2025/convind/data"
)

func getLinks(source io.Reader) ([]string, error) {
	text_, err := io.ReadAll(source)
	if err != nil {
		return nil, err
	}
	p := goldmark.DefaultParser()
	n := p.Parse(text.NewReader(text_))
	links := make([]string, 0)
	links = walkLinks(links, n)
	return links, nil
}

func walkLinks(links []string, n ast.Node) []string {
	switch n := n.(type) {
	case *ast.Link:
		dest := n.Destination
		links = append(links, string(dest))
	}
	if n.HasChildren() {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			links = walkLinks(links, c)
		}
	}
	return links
}

type HopsClass struct {
	dataStore data.DataStore

	aList [][2]data.ID
}

func NewHopsClass(dataStore data.DataStore) *HopsClass {
	return &HopsClass{
		dataStore: dataStore,
	}
}

func (c *HopsClass) Load() error {
	ids, err := c.dataStore.AllIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		d, err := c.dataStore.GetDataByID(id)
		if err != nil {
			return err
		}
		page := Page{d}
		pr, err := page.LatestRevision()
		if err != nil {
			return err
		}
		if pr == nil {
			continue
		}
		rc, err := pr.DataRevision.NewReadCloser()
		if err != nil {
			return err
		}
		links, err := getLinks(rc)
		if err != nil {
			return err
		}
		for _, link := range links {
			if !strings.HasPrefix(link, "convind://") {
				continue
			}
			id2, err := data.ParseID(link[10:])
			if err != nil {
				continue
			}
			c.aList = append(c.aList, [2]data.ID{id, id2})
		}
	}
	return nil
}

func (c *HopsClass) AttemptInstance(dr data.DataRevision) (data.Instance, error) {
	i := HopsInstance{class: c}
	for _, pair := range c.aList {
		if pair[1] == dr.Data().ID() {
			i.backHop1 = append(i.backHop1, pair[0])
		}
		if pair[0] == dr.Data().ID() {
			i.hop1 = append(i.backHop1, pair[1])
		}
	}
	// sort.Slice(i.hop1, func(j, k int) bool {
	// 	return i.hop1[j].String() < i.hop1[k].String()
	// })
	for _, pair := range c.aList {
		if !slices.ContainsFunc(i.hop1, func(a data.ID) bool { return a.String() == pair[0].String() }) {
			continue
		}
		i.hop2 = append(i.hop2, pair[1])
	}
	return &i, nil
}

type HopsInstance struct {
	class    *HopsClass
	dr       data.DataRevision
	backHop1 []data.ID
	hop1     []data.ID
	hop2     []data.ID
}

func (i *HopsInstance) DataRevision() data.DataRevision {
	return i.dr
}

func (i *HopsInstance) NewReadCloser() (io.ReadCloser, error) {
	data := map[string][]data.ID{"-1": i.backHop1, "1": i.hop1, "2": i.hop2}
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(data)
	if err != nil {
		return nil, err
	}
	return &buffer{buf}, nil
}

type buffer struct {
	*bytes.Buffer
}

func (b *buffer) Close() error { return nil }
