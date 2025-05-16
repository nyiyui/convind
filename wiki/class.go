package wiki

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
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
	case *ast.Image:
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

type WikiClass struct {
	dataStore data.DataStore

	aList  [][2]data.ID
	titles map[data.ID]string
}

func NewWikiClass(dataStore data.DataStore) *WikiClass {
	return &WikiClass{
		dataStore: dataStore,
	}
}

func (c *WikiClass) Name() string {
	return "inaba.kiyuri.ca/2025/convind/wiki"
}

func (c *WikiClass) Load() error {
	log.Printf("WikiClass.Load")
	ids, err := c.dataStore.AllIDs()
	if err != nil {
		return err
	}
	c.titles = map[data.ID]string{}
	for _, id := range ids {
		d, err := c.dataStore.GetDataByID(id)
		if err != nil {
			return err
		}
		if d.MIMEType() != "text/markdown" {
			continue
		}
		page := Page{d}
		pr, err := page.LatestRevision()
		if err != nil {
			return err
		}
		if pr == nil {
			continue
		}
		c.titles[id], err = pr.Title()
		if err != nil {
			return err
		}
		rc, err := pr.DataRevision.NewReadCloser()
		if err != nil {
			return err
		}
		links, err := getLinks(rc)
		if err != nil {
			return err
		}
		log.Printf("links from %s: %v", c.titles[id], links)
		for _, link := range links {
			if strings.HasPrefix(link, "convind://") {
				id2, err := data.ParseID(link[10:])
				if err != nil {
					continue
				}
				c.aList = append(c.aList, [2]data.ID{id, id2})
			} else if strings.HasPrefix(link, "/api/v1/data/") {
				id2, err := data.ParseID(link[13:])
				if err != nil {
					continue
				}
				c.aList = append(c.aList, [2]data.ID{id, id2})
			}
		}
	}
	return nil
}

func (c *WikiClass) AttemptInstance(dr data.DataRevision) (data.Instance, error) {
	// non text/markdown files may be linked to
	i := WikiInstance{class: c, title: c.titles[dr.Data().ID()]}
	for _, pair := range c.aList {
		if pair[0] == dr.Data().ID() {
			i.hop1 = append(i.hop1, pair[1])
		} else if pair[1] == dr.Data().ID() {
			i.hop1 = append(i.hop1, pair[0])
		}
	}
	// sort.Slice(i.hop1, func(j, k int) bool {
	// 	return i.hop1[j].String() < i.hop1[k].String()
	// })
	for _, pair := range c.aList {
		if slices.ContainsFunc(i.hop1, func(a data.ID) bool { return a.String() == pair[0].String() }) {
			i.hop2 = append(i.hop2, pair[1])
		}
		if slices.ContainsFunc(i.hop1, func(a data.ID) bool { return a.String() == pair[1].String() }) {
			i.hop2 = append(i.hop2, pair[0])
		}
	}
	return &i, nil
}

type WikiInstance struct {
	class *WikiClass
	dr    data.DataRevision
	title string
	hop1  []data.ID
	hop2  []data.ID
}

func (i *WikiInstance) DataRevision() data.DataRevision {
	return i.dr
}

func (i *WikiInstance) MIMEType() string { return "application/json" }

type pageEntry struct {
	ID    data.ID
	Title string
}

func (i *WikiInstance) NewReadCloser() (io.ReadCloser, error) {
	hop1 := make([]pageEntry, len(i.hop1))
	for j := range i.hop1 {
		hop1[j] = pageEntry{i.hop1[j], i.class.titles[i.hop1[j]]}
	}
	hop2 := make([]pageEntry, len(i.hop2))
	for j := range i.hop2 {
		hop2[j] = pageEntry{i.hop2[j], i.class.titles[i.hop2[j]]}
	}
	data := map[string]interface{}{"1": hop1, "2": hop2, "title": i.title}
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
