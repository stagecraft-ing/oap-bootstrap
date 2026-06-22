package cluster

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// hetznerSubpath is where the wrapped setup.sh and its kubeconfig live inside
// the forked repo. It is a stable part of the upstream layout the CLI couples to
// (spec 001 OQ-A): setup.sh hardcodes ENV_FILE and KUBECONFIG_PATH relative to
// its own directory, so the bridge .env and the captured kubeconfig are siblings
// of the script.
const hetznerSubpath = "platform/infra/hetzner"

// HetznerDir is the directory holding setup.sh inside a forked-repo checkout.
func HetznerDir(repoDir string) string { return filepath.Join(repoDir, hetznerSubpath) }

// BridgeEnvPath is where setup.sh sources its .env (it hardcodes
// "$SCRIPT_DIR/.env"); the bridge must materialise the cleartext env there.
func BridgeEnvPath(repoDir string) string { return filepath.Join(HetznerDir(repoDir), ".env") }

// KubeconfigPath is where setup.sh writes the cluster kubeconfig
// ("$SCRIPT_DIR/kubeconfig").
func KubeconfigPath(repoDir string) string { return filepath.Join(HetznerDir(repoDir), "kubeconfig") }

// EnsureCheckout makes a local checkout of the forked repo available at repoDir,
// cloning it if absent and fetching+fast-forwarding the default branch if it
// already exists. The cluster phase needs the fork's setup.sh, cluster.yaml,
// post-create.sh, and gitops tree on disk; the fork is public (AGPL), so an
// anonymous clone over HTTPS is sufficient. A pre-existing non-git directory is
// an error rather than a silent reuse.
func EnsureCheckout(ctx context.Context, repoDir, org, repo string) error {
	gitDir := filepath.Join(repoDir, ".git")
	if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
		// Existing checkout: fetch + fast-forward so setup.sh and the gitops tree
		// reflect the fork's current default branch. A pull failure is fatal: a
		// stale wrapper script is exactly the drift the cluster phase must avoid.
		if err := run(ctx, repoDir, "git", "fetch", "--quiet", "origin"); err != nil {
			return fmt.Errorf("git fetch in %s: %w", repoDir, err)
		}
		if err := run(ctx, repoDir, "git", "pull", "--quiet", "--ff-only"); err != nil {
			return fmt.Errorf("git pull --ff-only in %s: %w", repoDir, err)
		}
		return nil
	} else if err == nil || os.IsNotExist(err) {
		if _, statErr := os.Stat(repoDir); statErr == nil {
			return fmt.Errorf("%s exists but is not a git checkout; remove it or pass --repo-dir to an empty path", repoDir)
		}
	}
	url := fmt.Sprintf("https://github.com/%s/%s.git", org, repo)
	if err := run(ctx, "", "git", "clone", "--quiet", url, repoDir); err != nil {
		return fmt.Errorf("git clone %s: %w (the fork must be reachable; create it with `oap-bootstrap github` first)", url, err)
	}
	return nil
}

// RunSetupPhase1 runs the forked repo's setup.sh in the hetzner dir, streaming
// its output to the operator. setup.sh detects its own phase from .env state:
// with the identity-phase OIDC/Rauthy values still absent it provisions the
// cluster (hetzner-k3s create), bootstraps Flux, materialises the Phase-1
// secrets, prints the node IP, and exits 0 at its "Phase 1 Complete" gate. The
// CLI passes through the environment plus an explicit SOPS_AGE_KEY_FILE so the
// `flux bootstrap` sops-age Secret resolves the operator key on macOS, where
// sops's default key location differs from the upstream convention.
func RunSetupPhase1(ctx context.Context, repoDir, sopsAgeKeyFile string) error {
	dir := HetznerDir(repoDir)
	script := filepath.Join(dir, "setup.sh")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("setup.sh not found at %s: %w", script, err)
	}
	cmd := exec.CommandContext(ctx, "bash", "setup.sh")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	if sopsAgeKeyFile != "" {
		cmd.Env = append(cmd.Env, "SOPS_AGE_KEY_FILE="+sopsAgeKeyFile)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("setup.sh phase 1: %w", err)
	}
	return nil
}

// CaptureNodeIP reads the worker node's ExternalIP from the freshly-created
// cluster, mirroring the jsonpath setup.sh itself uses (a non-master node first,
// falling back to any node). It returns an error rather than the script's
// "pending" sentinel so a not-yet-assigned IP fails the phase loudly instead of
// persisting a bogus NODE_IP.
func CaptureNodeIP(ctx context.Context, kubeconfigPath string) (string, error) {
	const jsonpath = `jsonpath={.items[0].status.addresses[?(@.type=="ExternalIP")].address}`
	tries := [][]string{
		{"get", "nodes", "-l", "!node-role.kubernetes.io/master", "-o", jsonpath},
		{"get", "nodes", "-o", jsonpath},
	}
	var lastErr error
	for _, args := range tries {
		full := append([]string{"--kubeconfig", kubeconfigPath}, args...)
		cmd := exec.CommandContext(ctx, "kubectl", full...)
		cmd.Env = os.Environ()
		out, err := cmd.Output()
		if err != nil {
			lastErr = err
			continue
		}
		if ip := strings.TrimSpace(string(out)); ip != "" {
			return ip, nil
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("kubectl get nodes: %w", lastErr)
	}
	return "", fmt.Errorf("no node ExternalIP found via %s (cluster may still be provisioning)", kubeconfigPath)
}

// run executes name+args in dir (cwd if dir is empty), inheriting stdio so git
// progress and errors reach the operator.
func run(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}
