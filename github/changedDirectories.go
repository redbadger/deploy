package github

import (
	"fmt"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func appendIfMissing(slice []string, s string) []string {
	for _, ele := range slice {
		if ele == s {
			return slice
		}
	}
	return append(slice, s)
}

func getTopLevelDirName(path string) string {
	const separator = "/"
	if !strings.Contains(path, separator) {
		return ""
	}
	return strings.Split(path, separator)[0]
}

func getTree(repo *git.Repository, ref string) (tree *object.Tree, err error) {
	hash := plumbing.NewHash(ref)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		err = fmt.Errorf("Cannot get commit from hash %v: %v", hash, err)
		return
	}
	tree, err = commit.Tree()
	if err != nil {
		err = fmt.Errorf("Cannot get tree from commit %v: %v", commit, err)
		return
	}
	return
}

// GetChangedDirectories returns an array of unique top level directory names
// in which there have been changes
func GetChangedDirectories(repo *git.Repository, headRef, baseRef string) (directories []string, err error) {
	headTree, err := getTree(repo, headRef)
	if err != nil {
		err = fmt.Errorf("Cannot get tree from commit %v: %v", headRef, err)
		return
	}
	baseTree, err := getTree(repo, baseRef)
	if err != nil {
		err = fmt.Errorf("Cannot get tree from commit %v: %v", headRef, err)
		return
	}
	diff, err := headTree.Diff(baseTree)
	for _, change := range diff {
		name := change.To.Name
		if name == "" {
			name = change.From.Name
		}
		dir := getTopLevelDirName(name)
		if dir != "" {
			directories = appendIfMissing(directories, dir)
		}
	}
	return
}
