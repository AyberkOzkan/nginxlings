// Package packages installs the built Linux packages in the official image of
// each supported distribution and checks the installed program: it finds its
// exercises from any directory, creates the learner's ~/nginxlings, runs a
// host-NGINX exercise as root and as a regular user with the distribution's
// default PATH, and its setup check finds every command the exercises need.
// Debian- and Fedora-like systems install the .deb or .rpm, Arch the
// .pkg.tar.zst; Alpine, Void and Ubuntu (again) use install.sh from the
// release archive.
//
// Build the packages first, then point NGINXLINGS_PACKAGES at GoReleaser's
// dist directory:
//
//	goreleaser release --snapshot --clean
//	NGINXLINGS_PACKAGES=$PWD/dist go test ./tests/packages -v -timeout 40m
//
// Without the variable the tests are skipped. They need Docker and network
// access to pull images and packages.
package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"nginxlings/tests/testutil"
)

const packagesEnv = "NGINXLINGS_PACKAGES"

// arch names this machine's architecture the way each package format does.
type arch struct{ deb, rpm, pacman, tarball string }

func hostArch(t *testing.T) arch {
	switch runtime.GOARCH {
	case "amd64":
		return arch{deb: "amd64", rpm: "x86_64", pacman: "x86_64", tarball: "amd64"}
	case "arm64":
		return arch{deb: "arm64", rpm: "aarch64", pacman: "aarch64", tarball: "arm64"}
	}
	t.Skipf("no packages are built for %s", runtime.GOARCH)
	return arch{}
}

// distributions lists the images and how each installs nginxlings from /dist.
func distributions(a arch) []struct{ name, image, install string } {
	tarball := fmt.Sprintf("mkdir /x && tar -xzf /dist/nginxlings_*_linux_%s.tar.gz -C /x && /x/install.sh >/dev/null", a.tarball)
	apt := fmt.Sprintf("apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq /dist/nginxlings_*_%s.deb >/dev/null", a.deb)
	return []struct{ name, image, install string }{
		{"ubuntu", "ubuntu:24.04", apt},
		{"debian", "debian:12", apt},
		{"fedora", "fedora:latest", fmt.Sprintf("dnf install -y -q /dist/nginxlings-*.%s.rpm", a.rpm)},
		{"rocky", "rockylinux:9", fmt.Sprintf("dnf install -y -q /dist/nginxlings-*.%s.rpm", a.rpm)},
		{"opensuse", "opensuse/tumbleweed:latest", fmt.Sprintf("zypper -q --non-interactive --no-gpg-checks install --allow-unsigned-rpm /dist/nginxlings-*.%s.rpm", a.rpm)},
		{"arch", "archlinux:latest", fmt.Sprintf("pacman -Sy --noconfirm --quiet >/dev/null && pacman -U --noconfirm /dist/nginxlings-*-%s.pkg.tar.zst >/dev/null", a.pacman)},
		{"alpine", "alpine:latest", "apk add -q curl >/dev/null && " + tarball},
		{"void", "ghcr.io/void-linux/void-glibc-full:latest", "xbps-install -Syu -y xbps >/dev/null 2>&1; xbps-install -Sy -y curl >/dev/null && " + tarball},
		{"ubuntu-install-script", "ubuntu:24.04", tarball},
	}
}

func packagesDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(packagesEnv)
	if dir == "" {
		t.Skipf("%s is not set; see the package comment or tests/TESTING.md", packagesEnv)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not installed")
	}
	return abs
}

// shippedExercises counts the exercises in the repository's manifest.
func shippedExercises(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(testutil.RepoRoot(t), "exercises", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Exercises []json.RawMessage `json:"exercises"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return len(manifest.Exercises)
}

func TestPackagesInstallOnEachDistribution(t *testing.T) {
	dist := packagesDir(t)
	a := hostArch(t)
	verify := filepath.Join(testutil.RepoRoot(t), "tests", "packages", "verify.sh")
	expected := fmt.Sprint(shippedExercises(t))
	for _, d := range distributions(a) {
		d := d
		t.Run(d.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"-v", dist+":/dist:ro", "-v", verify+":/verify.sh:ro", "-e", "EXPECTED_EXERCISES="+expected,
				d.image, "sh", "-c", d.install+" && sh /verify.sh")
			out, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(out), "VERIFY OK") {
				t.Fatalf("%s: %v\n%s", d.image, err, tail(string(out), 40))
			}
			t.Log(tail(string(out), 12))
		})
	}
}

// podmanFailures are the Docker exercises that cannot pass with Podman's
// docker command: 10 points NGINX at Docker's embedded DNS on 127.0.0.11.
var podmanFailures = map[string]bool{"10_upstream_status_zone": true}

// TestPodmanRunsTheDockerExercises installs the .rpm on Rocky Linux, which
// brings Podman instead of Docker, and runs every Docker exercise from its
// starting files and with its solution. It also needs NGINXLINGS_SOLUTIONS
// and NGINXLINGS_PODMAN=1, because it runs a privileged container for about
// half an hour.
func TestPodmanRunsTheDockerExercises(t *testing.T) {
	dist := packagesDir(t)
	a := hostArch(t)
	solutions := os.Getenv("NGINXLINGS_SOLUTIONS")
	if solutions == "" || os.Getenv("NGINXLINGS_PODMAN") != "1" {
		t.Skip("set NGINXLINGS_SOLUTIONS and NGINXLINGS_PODMAN=1 to run the Podman exercises")
	}
	script := filepath.Join(testutil.RepoRoot(t), "tests", "packages", "podman.sh")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "run", "--rm", "--privileged",
		"-v", dist+":/dist:ro", "-v", solutions+":/solutions:ro", "-v", script+":/podman.sh:ro",
		"-e", "RPM_ARCH="+a.rpm, "rockylinux:9", "sh", "/podman.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("podman run: %v\n%s", err, tail(string(out), 40))
	}
	checked := 0
	for _, line := range strings.Split(string(out), "\n") {
		ok, okLine := strings.CutPrefix(line, "OK ")
		bad, badLine := strings.CutPrefix(line, "BAD ")
		switch {
		case okLine:
			checked++
			if podmanFailures[ok] {
				t.Errorf("%s now passes with Podman; remove it from podmanFailures", ok)
			}
		case badLine:
			checked++
			id, _, _ := strings.Cut(bad, ":")
			if !podmanFailures[id] {
				t.Errorf("%s", line)
			}
		}
	}
	if checked == 0 {
		t.Fatalf("no exercise was checked:\n%s", tail(string(out), 40))
	}
	t.Logf("%d Docker exercises checked with Podman", checked)
}

func tail(s string, lines int) string {
	all := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.Join(all, "\n")
}
