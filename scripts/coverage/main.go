package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	packages, err := listPackages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "listing packages: %v\n", err)
		os.Exit(1)
	}

	filtered := filterPackages(packages)
	if len(filtered) == 0 {
		fmt.Fprintln(os.Stderr, "no packages selected for coverage")
		os.Exit(1)
	}

	if err := run("go", append([]string{"test", "-coverprofile=coverage.out"}, filtered...)...); err != nil {
		fmt.Fprintf(os.Stderr, "running coverage tests: %v\n", err)
		os.Exit(1)
	}

	if err := run("go", "tool", "cover", "-func=coverage.out"); err != nil {
		fmt.Fprintf(os.Stderr, "rendering coverage report: %v\n", err)
		os.Exit(1)
	}
}

func listPackages() ([]string, error) {
	cmd := exec.Command("go", "list", "./...")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := bytes.Split(bytes.TrimSpace(out), []byte{'\n'})
	packages := make([]string, 0, len(lines))
	for _, line := range lines {
		pkg := strings.TrimSpace(string(line))
		if pkg != "" {
			packages = append(packages, pkg)
		}
	}
	return packages, nil
}

func filterPackages(packages []string) []string {
	filtered := make([]string, 0, len(packages))
	for _, pkg := range packages {
		switch {
		case strings.Contains(pkg, "/cmd/api"):
			continue
		case strings.Contains(pkg, "/scripts/"):
			continue
		case strings.HasSuffix(pkg, "/internal/application/ports"):
			continue
		default:
			filtered = append(filtered, pkg)
		}
	}
	return filtered
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
