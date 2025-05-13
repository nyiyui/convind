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

func (c *WikiClass) Load() error {
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
		log.Printf("title %s", c.titles[id])
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

func (c *WikiClass) AttemptInstance(dr data.DataRevision) (data.Instance, error) {
	i := WikiInstance{class: c, title: c.titles[dr.Data().ID()]}
	for _, pair := range c.aList {
		if pair[1] == dr.Data().ID() {
			i.hop1Back = append(i.hop1Back, pair[0])
		}
		if pair[0] == dr.Data().ID() {
			i.hop1 = append(i.hop1Back, pair[1])
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

type WikiInstance struct {
	class    *WikiClass
	dr       data.DataRevision
	title    string
	hop1Back []data.ID
	hop1     []data.ID
	hop2     []data.ID
}

func (i *WikiInstance) DataRevision() data.DataRevision {
	return i.dr
}

type pageEntry struct {
	ID    data.ID
	Title string
}

func (i *WikiInstance) NewReadCloser() (io.ReadCloser, error) {
	hop1Back := make([]pageEntry, len(i.hop1Back))
	for j := range i.hop1Back {
		hop1Back[j] = pageEntry{i.hop1Back[j], i.class.titles[i.hop1Back[j]]}
	}
	hop1 := make([]pageEntry, len(i.hop1))
	for j := range i.hop1 {
		hop1[j] = pageEntry{i.hop1[j], i.class.titles[i.hop1[j]]}
	}
	hop2 := make([]pageEntry, len(i.hop2))
	for j := range i.hop2 {
		hop2[j] = pageEntry{i.hop2[j], i.class.titles[i.hop2[j]]}
	}
	data := map[string]interface{}{"-1": hop1Back, "1": hop1, "2": hop2, "title": i.title}
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
