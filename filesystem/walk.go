package filesystem

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-billy.v4"
)

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(fs billy.Filesystem, dirname string) ([]os.FileInfo, error) {
	names, err := fs.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	// sort.Strings(names)
	return names, nil
}

// ErrSkipDir is used as a return value from WalkFuncs to indicate that
// the directory named in the call is to be skipped. It is not returned
// as an error by any function.
var ErrSkipDir = errors.New("skip this directory")

// WalkFunc is the type of the function called for each file or directory
// visited by Walk. The path argument contains the argument to Walk as a
// prefix; that is, if Walk is called with "dir", which is a directory
// containing the file "a", the walk function will be called with argument
// "dir/a". The info argument is the os.FileInfo for the named path.
//
// If there was a problem walking to the file or directory named by path, the
// incoming error will describe the problem and the function can decide how
// to handle that error (and Walk will not descend into that directory). If
// an error is returned, processing stops. The sole exception is when the function
// returns the special value ErrSkipDir. If the function returns ErrSkipDir when invoked
// on a directory, Walk skips the directory's contents entirely.
// If the function returns ErrSkipDir when invoked on a non-directory file,
// Walk skips the remaining files in the containing directory.
type WalkFunc func(fs billy.Filesystem, path string, info os.FileInfo, err error) error

// walk recursively descends path, calling walkFn.
func walk(fs billy.Filesystem, path string, info os.FileInfo, walkFn WalkFunc) error {
	if !info.IsDir() {
		return walkFn(fs, path, info, nil)
	}

	infos, err := readDirNames(fs, path)
	err1 := walkFn(fs, path, info, err)
	// If err != nil, walk can't walk into this directory.
	// err1 != nil means walkFn want walk to skip this directory or stop walking.
	// Therefore, if one of err and err1 isn't nil, walk will return.
	if err != nil || err1 != nil {
		// The caller's behavior is controlled by the return value, which is decided
		// by walkFn. walkFn may ignore err and return nil.
		// If walkFn returns ErrSkipDir, it will be handled by the caller.
		// So walk should return whatever walkFn returns.
		return err1
	}

	for _, info := range infos {
		filename := filepath.Join(path, info.Name())
		fileInfo, err := fs.Lstat(filename)
		if err != nil {
			if err := walkFn(fs, filename, fileInfo, err); err != nil && err != ErrSkipDir {
				return err
			}
		} else {
			err = walk(fs, filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != ErrSkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
func Walk(fs billy.Filesystem, root string, walkFn WalkFunc) error {
	info, err := fs.Lstat(root)
	if err != nil {
		err = walkFn(fs, root, nil, err)
	} else {
		err = walk(fs, root, info, walkFn)
	}
	if err == ErrSkipDir {
		return nil
	}
	return err
}
