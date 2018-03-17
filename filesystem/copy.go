package filesystem

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	billy "gopkg.in/src-d/go-billy.v4"
)

// Copy copies src on the local filesystem to dest, in the destFs filesystem
func Copy(src, dest string, destFs billy.Filesystem) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return copy(src, dest, info, destFs)
}

func copy(src, dest string, info os.FileInfo, destFs billy.Filesystem) error {
	if info.IsDir() {
		return dcopy(src, dest, info, destFs)
	}
	return fcopy(src, dest, info, destFs)
}

func fcopy(src, dest string, info os.FileInfo, destFs billy.Filesystem) error {

	f, err := destFs.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = io.Copy(f, s)
	return err
}

func dcopy(src, dest string, info os.FileInfo, destFs billy.Filesystem) error {

	if err := destFs.MkdirAll(dest, info.Mode()); err != nil {
		return err
	}

	infos, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, info := range infos {
		if err := copy(
			filepath.Join(src, info.Name()),
			filepath.Join(dest, info.Name()),
			info,
			destFs,
		); err != nil {
			return err
		}
	}

	return nil
}
