package filesystem

import (
	"fmt"
	"path"

	"gopkg.in/src-d/go-billy.v4"
)

// Remove recursively removes the specified file/directory
func Remove(file string, fs billy.Filesystem) (err error) {
	info, err := fs.Lstat(file)
	if err != nil {
		return
	}
	if info.IsDir() {
		infos, err := fs.ReadDir(file)
		for _, info1 := range infos {
			name := path.Join(file, info1.Name())
			err = Remove(name, fs)
			if err != nil {
				return err
			}
		}
	}
	err = fs.Remove(file)
	if err != nil {
		return fmt.Errorf("error removing %s", file)
	}
	return
}
