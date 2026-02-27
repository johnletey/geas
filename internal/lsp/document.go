package lsp

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fjl/geas/internal/ast"
)

// documentStore manages open .eas files.
type documentStore struct {
	mu   sync.Mutex
	docs map[string]*document // keyed by URI
}

// document represents an open .eas file.
type document struct {
	URI     string
	Path    string // filesystem path derived from URI
	Version int
	Content string

	// Parsed state (updated on each change).
	AST    *ast.Document
	Errors []*ast.ParseError
}

func newDocumentStore() *documentStore {
	return &documentStore{docs: make(map[string]*document)}
}

func (s *documentStore) open(uri string, version int, content string) *document {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := &document{
		URI:     uri,
		Path:    uriToPath(uri),
		Version: version,
		Content: content,
	}
	doc.parse()
	s.docs[uri] = doc
	return doc
}

func (s *documentStore) change(uri string, version int, content string) *document {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.docs[uri]
	if !ok {
		return nil
	}
	doc.Version = version
	doc.Content = content
	doc.parse()
	return doc
}

func (s *documentStore) close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

func (s *documentStore) get(uri string) *document {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.docs[uri]
}

// overlayFS returns an fs.FS that reads open documents from the store,
// falling back to the real filesystem for files not in the store.
func (s *documentStore) overlayFS(base string) fs.FS {
	return &overlayFS{store: s, base: base}
}

func (d *document) parse() {
	p := ast.NewParser(d.Path, []byte(d.Content))
	d.AST, d.Errors = p.Parse()
}

// overlayFS is an fs.FS that overlays open documents on the real filesystem.
type overlayFS struct {
	store *documentStore
	base  string
}

func (f *overlayFS) Open(name string) (fs.File, error) {
	// Check if any open document matches this path.
	f.store.mu.Lock()
	for _, doc := range f.store.docs {
		relPath, err := filepath.Rel(f.base, doc.Path)
		if err == nil && relPath == name {
			f.store.mu.Unlock()
			return &memFile{
				name:    name,
				content: []byte(doc.Content),
			}, nil
		}
	}
	f.store.mu.Unlock()

	// Fall back to real filesystem.
	return os.DirFS(f.base).Open(name)
}

// memFile is an in-memory fs.File.
type memFile struct {
	name    string
	content []byte
	offset  int
}

func (f *memFile) Read(b []byte) (int, error) {
	if f.offset >= len(f.content) {
		return 0, fs.ErrNotExist
	}
	n := copy(b, f.content[f.offset:])
	f.offset += n
	return n, nil
}

func (f *memFile) Stat() (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func (f *memFile) Close() error {
	return nil
}

// uriToPath converts a file:// URI to a filesystem path.
func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		// Fallback: strip file:// prefix.
		return strings.TrimPrefix(uri, "file://")
	}
	return u.Path
}

// pathToURI converts a filesystem path to a file:// URI.
func pathToURI(path string) string {
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
	}
	return "file://" + path
}
