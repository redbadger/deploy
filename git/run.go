package git

import (
	"bytes"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// Run executes git (in the specified workingDir) with args
// and returns the captured stdout
func Run(workingDir string, args ...string) (stdout, stderr string, err error) {
	log.Info("git", args)
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	cmd.Dir = workingDir
	var o, e bytes.Buffer
	cmd.Stderr = &e
	cmd.Stdout = &o
	err = cmd.Run()
	stdout = o.String()
	stderr = e.String()
	return
}

// MustRun executes git (in the specified workingDir) with args.
// Fatal if there is an error.
func MustRun(workingDir string, args ...string) {
	o, e, err := Run(workingDir, args...)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"stdout": o,
			"stderr": e,
		}).Fatal()
	}
}
