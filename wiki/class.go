package wiki

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"inaba.kiyuri.ca/2025/convind/data"
)

func getLinks(source io.Reader) ([]link, error) {
	text_, err := io.ReadAll(source)
	if err != nil {
		return nil, err
	}
	p := goldmark.DefaultParser()
	n := p.Parse(text.NewReader(text_))
	links := make([]link, 0)
	links = walkLinks(links, n, text_)
	return links, nil
}

type link struct {
	Destination string
	// Content is the text surrounding the link itself.
	Context string
}

func walkLinks(links []link, n ast.Node, source []byte) []link {
	switch n := n.(type) {
	case *ast.Link:
		dest := n.Destination
		context := getTextContent(n, source)
		links = append(links, link{
			Destination: string(dest),
			Context:     context,
		})
	case *ast.Image:
		dest := n.Destination
		context := getTextContent(n, source)
		links = append(links, link{
			Destination: string(dest),
			Context:     context,
		})
	}
	if n.HasChildren() {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			links = walkLinks(links, c, source)
		}
	}
	return links
}

// Helper function to extract text content from a node
func getTextContent(n ast.Node, source []byte) string {
	if n == nil {
		return ""
	}

	// Build context from surrounding text
	var context strings.Builder

	// Get the parent node to find siblings
	parent := n.Parent()
	if parent == nil {
		// If no parent, just get the node's own text
		return extractNodeText(n, source)
	}

	// Find the current node among its siblings
	var prevSibling, nextSibling ast.Node
	var foundCurrent bool

	// Traverse siblings to find current node, previous, and next
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if child == n {
			foundCurrent = true
		} else if !foundCurrent {
			prevSibling = child // This will be the last sibling before current
		} else if foundCurrent && nextSibling == nil {
			nextSibling = child // This will be the first sibling after current
			break
		}
	}

	// Extract text from previous sibling (if exists)
	if prevSibling != nil {
		prevText := truncateText(extractNodeText(prevSibling, source), 30)
		if prevText != "" {
			context.WriteString(prevText)
			context.WriteString(" … ")
		}
	}

	// Extract text from current node
	nodeText := extractNodeText(n, source)
	context.WriteString(nodeText)

	// Extract text from next sibling (if exists)
	if nextSibling != nil {
		nextText := truncateText(extractNodeText(nextSibling, source), 30)
		if nextText != "" {
			context.WriteString(" … ")
			context.WriteString(nextText)
		}
	}

	return context.String()
}

// Helper function to extract text directly from a node
func extractNodeText(n ast.Node, source []byte) string {
	if n == nil {
		return ""
	}

	var sb strings.Builder

	// If it's a text node, extract its content
	if text, ok := n.(*ast.Text); ok {
		if text.Segment.Start < len(source) && text.Segment.Stop <= len(source) && text.Segment.Start < text.Segment.Stop {
			sb.Write(text.Segment.Value(source))
		}
		return sb.String()
	}

	// Otherwise recursively extract text from children
	if n.HasChildren() {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if text, ok := c.(*ast.Text); ok {
				if text.Segment.Start < len(source) && text.Segment.Stop <= len(source) && text.Segment.Start < text.Segment.Stop {
					sb.Write(text.Segment.Value(source))
				}
			} else {
				sb.WriteString(extractNodeText(c, source))
			}
		}
	}

	return sb.String()
}

// Truncate text to a maximum length
func truncateText(text string, maxLength int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLength {
		return text
	}

	// For very long texts, take the last part which is likely more relevant to the link
	return text[len(text)-maxLength:]
}

type WikiClass struct {
	dataStore data.DataStore

	aList              []wikiEdge
	titles             map[data.ID]string
	mimeTypes          map[data.ID]string
	latestCreationTime time.Time
}

type wikiEdge struct {
	Src        data.ID
	Dst        data.ID
	SrcContext string
}

func NewWikiClass(dataStore data.DataStore) *WikiClass {
	return &WikiClass{
		dataStore: dataStore,
	}
}

func (c *WikiClass) Name() string {
	return "inaba.kiyuri.ca/2025/convind/wiki"
}

func (c *WikiClass) getLatestCreationTime() (time.Time, error) {
	var latestCreationTime time.Time
	ids, err := c.dataStore.AllIDs()
	if err != nil {
		return time.Time{}, nil
	}
	for _, id := range ids {
		d, err := c.dataStore.GetDataByID(id)
		if err != nil {
			return time.Time{}, nil
		}
		dr, err := data.LatestRevision(d)
		if err != nil {
			return time.Time{}, nil
		}
		if dr == nil {
			continue
		}
		ct := dr.CreationTime()
		if ct.After(latestCreationTime) {
			latestCreationTime = ct
		}
	}
	return latestCreationTime, nil
}

func (c *WikiClass) ReloadIfOutdated() error {
	lct, err := c.getLatestCreationTime()
	if err != nil {
		return err
	}
	if lct.After(c.latestCreationTime) {
		c.latestCreationTime = lct
		return c.Load()
	}
	return nil
}

func (c *WikiClass) Load() error {
	log.Printf("WikiClass.Load")
	ids, err := c.dataStore.AllIDs()
	if err != nil {
		return err
	}
	c.titles = map[data.ID]string{}
	c.mimeTypes = map[data.ID]string{}
	for _, id := range ids {
		d, err := c.dataStore.GetDataByID(id)
		if err != nil {
			return err
		}
		c.mimeTypes[id] = d.MIMEType()
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
		slices.SortFunc(links, func(a, b link) int {
			return strings.Compare(a.Destination, b.Destination)
		})
		links = slices.CompactFunc(links, func(a, b link) bool {
			return a.Destination == b.Destination && a.Context == b.Context
		})
		for _, link := range links {
			if strings.HasPrefix(link.Destination, "convind://") {
				id2, err := data.ParseID(link.Destination[10:])
				if err != nil {
					continue
				}
				c.aList = append(c.aList, wikiEdge{
					Src:        id,
					Dst:        id2,
					SrcContext: link.Context,
				})
			} else if strings.HasPrefix(link.Destination, "/api/v1/data/") {
				id2, err := data.ParseID(link.Destination[13:])
				if err != nil {
					continue
				}
				c.aList = append(c.aList, wikiEdge{
					Src:        id,
					Dst:        id2,
					SrcContext: link.Context,
				})
			}
		}
	}
	return nil
}

func (c *WikiClass) AttemptInstance(dr data.DataRevision) (data.Instance, error) {
	c.ReloadIfOutdated()
	// non text/markdown files may be linked to
	i := WikiInstance{class: c, title: c.titles[dr.Data().ID()]}
	for _, edge := range c.aList {
		if edge.Src == dr.Data().ID() {
			i.hop1 = append(i.hop1, hopWithContext{edge.Dst, ""})
		} else if edge.Dst == dr.Data().ID() {
			i.hop1 = append(i.hop1, hopWithContext{edge.Src, edge.SrcContext})
		}
	}
	slices.SortFunc(i.hop1, func(a, b hopWithContext) int {
		return strings.Compare(a.ID.String(), b.ID.String())
	})
	i.hop1 = slices.CompactFunc(i.hop1, func(a, b hopWithContext) bool {
		return a.ID == b.ID && a.Context == b.Context
	})
	// sort.Slice(i.hop1, func(j, k int) bool {
	// 	return i.hop1[j].String() < i.hop1[k].String()
	// })
	for _, edge := range c.aList {
		if slices.ContainsFunc(i.hop1, func(h hopWithContext) bool { return h.ID.String() == edge.Src.String() }) {
			i.hop2 = append(i.hop2, edge.Dst)
		}
		if slices.ContainsFunc(i.hop1, func(h hopWithContext) bool { return h.ID.String() == edge.Dst.String() }) {
			i.hop2 = append(i.hop2, edge.Src)
		}
	}
	slices.SortFunc(i.hop2, func(a, b data.ID) int {
		return strings.Compare(a.String(), b.String())
	})
	i.hop2 = slices.CompactFunc(i.hop2, func(a, b data.ID) bool {
		return a.String() == b.String()
	})
	return &i, nil
}

type WikiInstance struct {
	class *WikiClass
	dr    data.DataRevision
	title string
	hop1  []hopWithContext
	hop2  []data.ID
}

type hopWithContext struct {
	data.ID
	Context string
}

func (i *WikiInstance) DataRevision() data.DataRevision {
	return i.dr
}

func (i *WikiInstance) MIMEType() string { return "application/json" }

type pageEntry struct {
	ID       data.ID
	Title    string
	Context  string
	MIMEType string
}

func (i *WikiInstance) NewReadCloser() (io.ReadCloser, error) {
	hop1 := make([]pageEntry, len(i.hop1))
	for j := range i.hop1 {
		hop1[j] = pageEntry{ID: i.hop1[j].ID, Title: i.class.titles[i.hop1[j].ID], Context: i.hop1[j].Context, MIMEType: i.class.mimeTypes[i.hop1[j].ID]}
	}
	hop2 := make([]pageEntry, len(i.hop2))
	for j := range i.hop2 {
		hop2[j] = pageEntry{ID: i.hop2[j], Title: i.class.titles[i.hop2[j]], MIMEType: i.class.mimeTypes[i.hop2[j]]}
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
