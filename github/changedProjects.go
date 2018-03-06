package github

import (
	"log"
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

func getTree(repo *git.Repository, ref string) (tree *object.Tree, err error) {
	hash := plumbing.NewHash(ref)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		log.Fatalf("Cannot get commit from hash %v: %v", hash, err)
	}
	tree, err = commit.Tree()
	if err != nil {
		log.Fatalf("Cannot get tree from commit %v: %v", commit, err)
	}
	return
}

// GetChangedProjects returns an array of unique top level directory names
// in which there have been changes
func GetChangedProjects(repo *git.Repository, headRef, baseRef string) (files []string, err error) {
	headTree, err := getTree(repo, headRef)
	if err != nil {
		return
	}
	baseTree, err := getTree(repo, baseRef)
	if err != nil {
		return
	}
	diff, err := headTree.Diff(baseTree)
	for _, change := range diff {
		files = appendIfMissing(files, strings.Split(change.To.Name, "/")[0])
	}
	return
}
