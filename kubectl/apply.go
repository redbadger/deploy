package kubectl

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

// Apply passes the supplied manifests to the stdin of `kubectl apply -f -`
func Apply(namespace, manifests string) (err error) {
	cmd := exec.Command("kubectl", "--namespace", namespace, "apply", "-f", "-")
	cmd.Env = append(os.Environ(),
		"X=foo",
	)
	cmd.Stdin = strings.NewReader(manifests)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return
}
