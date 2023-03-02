package docker

import (
	"fmt"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/internal/cli"
	"strings"
)

const fixedVolumeName = "spectro-system-cluster"

//name = system-cluster
func ensureVolume(name string, status *cli.Status,) error {
	status.Start(fmt.Sprintf("Ensuring SpectroCloud volume (%s) 💾", name))
	if exists, err := removeDuplicateVolumes(name);err != nil {
		return err
	} else if exists {
		return nil
	}

	if err := createVolume(name); err != nil {
		status.End(false)
		return err
	}
	status.End(true)
	return nil
}

func removeDuplicateVolumes(name string) (bool, error) {
	volumes, err := volumesWithName(name)
	if err != nil {
		return false, err
	}

	return len(volumes) > 0, nil
}

// networksWithName returns a list of network IDs for networks with this name
func volumesWithName(name string) ([]string, error) {
	lsOut, err := exec.Output(exec.Command(
		"docker", "volume", "ls",
		"--filter=name="+name,
		"--format={{.Name}}",
	))
	if err != nil {
		return nil, err
	}
	cleaned := strings.TrimSuffix(string(lsOut), "\n")
	if cleaned == "" { // avoid returning []string{""}
		return nil, nil
	}
	return strings.Split(cleaned, "\n"), nil
}

func createVolume(name string) error {
	args := []string{"volume", "create"}
	args = append(args, name)
	return exec.Command("docker", args...).Run()
}