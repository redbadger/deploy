package kubectl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Apply passes the supplied manifests to the stdin of `kubectl apply -f -`
func Apply(namespace, manifests string) (err error) {
	cmd := exec.Command("kubectl", "--namespace", namespace, "apply", "-f", "-")
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(manifests)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("Cannot run kubectl: %v", err)
		return
	}
	return
}
