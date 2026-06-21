// Package cli is the oap-bootstrap command surface. M1 ships the skeleton plus
// two working commands (init, doctor); the provisioning phases are registered as
// stubs so the command tree and `--help` are complete. Dispatch is hand-rolled
// on the standard library to keep the binary dependency-free; swapping in cobra
// later is mechanical. See spec 001 section 3.2 (Command surface).
package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
)

// Command is one subcommand.
type Command struct {
	Name  string
	Short string
	Run   func(args []string) error
}

// order is the display order in help.
var order = []string{
	"init", "doctor", "github", "cluster", "dns",
	"identity", "platform", "verify", "apply",
}

func commands() map[string]Command {
	return map[string]Command{
		"init":     {"init", "Collect/generate config into oap.env", cmdInit},
		"doctor":   {"doctor", "Preflight: required tools + config readiness", cmdDoctor},
		"github":   stub("github", "Fork + register the GitHub App + Actions secrets"),
		"cluster":  stub("cluster", "Wrap upstream setup.sh phase 1 (K3s + GitOps)"),
		"dns":      stub("dns", "Create Cloudflare A records; wait for certs"),
		"identity": stub("identity", "Create Rauthy OIDC clients (+ guided provider)"),
		"platform": stub("platform", "Wrap upstream setup.sh phase 2 (secrets + deploy)"),
		"verify":   stub("verify", "Endpoint / cert / webhook health report"),
		"apply":    stub("apply", "Run every phase (use with --yes)"),
	}
}

// Main parses args and dispatches. It returns a process exit code.
func Main(args []string) int {
	if len(args) < 1 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		usage()
		return 0
	}
	cmds := commands()
	c, ok := cmds[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "oap-bootstrap: unknown command %q\n\n", args[0])
		usage()
		return 2
	}
	if err := c.Run(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "oap-bootstrap %s: %v\n", c.Name, err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Println("oap-bootstrap: stand up an open-agentic-platform instance in a new org")
	fmt.Println("\nUsage:\n  oap-bootstrap <command> [flags]\n\nCommands:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	cmds := commands()
	for _, name := range order {
		fmt.Fprintf(w, "  %s\t%s\n", name, cmds[name].Short)
	}
	w.Flush()
	fmt.Println("\nRun 'oap-bootstrap <command> -h' for command flags.")
}

// stub builds a not-yet-implemented command that still documents itself.
func stub(name, short string) Command {
	return Command{Name: name, Short: short, Run: func(_ []string) error {
		return fmt.Errorf("phase %q is not implemented yet (M1 ships init + doctor); see spec 001 milestones", name)
	}}
}

// sortedStrings returns a sorted copy (small helper used by reports).
func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
