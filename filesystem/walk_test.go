package filesystem_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/redbadger/deploy/filesystem"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
)

type Node struct {
	name    string
	entries []*Node // nil if the entry is a file
	mark    int
}

var tree = &Node{
	"testdata",
	[]*Node{
		{"b", nil, 0},
		{"a", []*Node{}, 0},
		{"c", nil, 0},
		{
			"d",
			[]*Node{
				{"y", nil, 0},
				{"x", []*Node{}, 0},
				{
					"z",
					[]*Node{
						{"v", nil, 0},
						{"u", nil, 0},
					},
					0,
				},
			},
			0,
		},
	},
	0,
}

func walkTree(fs billy.Filesystem, n *Node, path string, f func(path string, n *Node)) {
	f(path, n)
	for _, e := range n.entries {
		walkTree(fs, e, filepath.Join(path, e.name), f)
	}
}

func makeTree(fs billy.Filesystem, t *testing.T) {
	walkTree(fs, tree, tree.name, func(path string, n *Node) {
		if n.entries == nil {
			fd, err := fs.Create(path)
			if err != nil {
				t.Errorf("makeTree: %v", err)
				return
			}
			fd.Close()
		} else {
			fs.MkdirAll(path, 0770)
		}
	})
}

// Assumes that each node name is unique. Good enough for a test.
// If clear is true, any incoming error is cleared before return. The errors
// are always accumulated, though.
func mark(fs billy.Filesystem, info os.FileInfo, err error, errors *[]error, clear bool) error {
	if info == nil {
		return nil
	}
	name := info.Name()
	walkTree(fs, tree, tree.name, func(path string, n *Node) {
		if n.name == name {
			n.mark++
		}
	})
	if err != nil {
		*errors = append(*errors, err)
		if clear {
			return nil
		}
		return err
	}
	return nil
}

func markTree(fs billy.Filesystem, n *Node) {
	walkTree(fs, n, "", func(path string, n *Node) { n.mark++ })
}

func checkMarks(fs billy.Filesystem, t *testing.T, report bool) {
	walkTree(fs, tree, tree.name, func(path string, n *Node) {
		if n.mark != 1 && report {
			t.Errorf("node %s mark = %d; expected 1", path, n.mark)
		}
		n.mark = 0
	})
}

func Test_Walk(t *testing.T) {
	fs := memfs.New()
	makeTree(fs, t)
	errors := make([]error, 0, 10)
	clear := true
	expectedFiles := []string{
		"testdata",
		"testdata/a",
		"testdata/b",
		"testdata/c",
		"testdata/d",
		"testdata/d/x",
		"testdata/d/y",
		"testdata/d/z",
		"testdata/d/z/u",
		"testdata/d/z/v",
	}
	files := []string{}
	markFn := func(fs billy.Filesystem, path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return mark(fs, info, err, &errors, clear)
	}
	// Expect no errors.
	err := filesystem.Walk(fs, tree.name, markFn)
	if err != nil {
		t.Fatalf("no error expected, found: %s", err)
	}
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %s", errors)
	}
	if !reflect.DeepEqual(files, expectedFiles) {
		t.Fatalf("got file list %v, want %v", files, expectedFiles)
	}
	checkMarks(fs, t, true)
	errors = errors[0:0]
}
