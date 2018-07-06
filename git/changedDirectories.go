package git

import (
	"fmt"
	"strings"
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

// GetChangedDirectories returns an array of unique top level directory names
// in which there have been changes
func GetChangedDirectories(srcDir, baseRef string) (directories []string, err error) {
	o, e, err := Run(srcDir, "diff", "--name-only", baseRef)
	if err != nil {
		return nil, fmt.Errorf("Error in git diff: %v (%s)", err, e)
	}
	diff := strings.Split(o, "\n")
	for _, change := range diff {
		dir := getTopLevelDirName(change)
		if dir != "" {
			directories = appendIfMissing(directories, dir)
		}
	}
	return
}
