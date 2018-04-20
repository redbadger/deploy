package kubectl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Apply passes the supplied manifests to the stdin of `kubectl apply -f -`
func Apply(namespace, manifests string, isDryRun bool) (output string, err error) {
	var args = []string{"--namespace", namespace, "apply", "-f", "-"}
	if isDryRun {
		args = append(args, "--dry-run")
	}
	cmd := exec.Command("kubectl", args...)
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(manifests)
	out, err := cmd.CombinedOutput()
	output = string(out)
	if err != nil {
		return output, fmt.Errorf("Cannot run kubectl: %v", err)
	}
	return
}
