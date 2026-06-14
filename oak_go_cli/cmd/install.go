package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/oakestra/oak-go-cli/cmd/install"
	"github.com/spf13/cobra"

	"github.com/oakestra/oak-go-cli/internal/api"
	"github.com/oakestra/oak-go-cli/internal/config"
)

// ─── top-level install command ────────────────────────────────────────────────

var installCmd = &cobra.Command{
	Use:   "install <root|cluster|worker|full> [version]",
	Short: "Install Oakestra components on this machine",
	RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// ─── flags ────────────────────────────────────────────────────────────────────

var (
	installRootYes    bool
	installClusterYes bool
	installWorkerYes  bool
	installFullYes    bool
	installSudo       bool
)

func init() {
	installCmd.AddCommand(installRootCmd)
	installCmd.AddCommand(installClusterCmd)
	installCmd.AddCommand(installWorkerCmd)
	installCmd.AddCommand(installFullCmd)

	// --root is inherited by all install subcommands.
	installCmd.PersistentFlags().BoolVar(&installSudo, "root", false,
		"Run install scripts with sudo")

	installRootCmd.Flags().BoolVarP(&installRootYes, "yes", "y", false, "Skip confirmation prompt")
	installClusterCmd.Flags().BoolVarP(&installClusterYes, "yes", "y", false, "Skip confirmation prompt")
	installWorkerCmd.Flags().BoolVarP(&installWorkerYes, "yes", "y", false, "Skip confirmation prompt")
	installFullCmd.Flags().BoolVarP(&installFullYes, "yes", "y", false, "Skip all confirmation prompts")
}

// ─── install root ─────────────────────────────────────────────────────────────

var installRootCmd = &cobra.Command{
	Use:   "root [version]",
	Short: "Install the Oakestra root orchestrator",
	Long: `Install the Oakestra root orchestrator on this machine.

Requires Docker with Compose support.
Optionally specify a version tag (e.g. v0.4.400).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doInstallRoot(firstArg(args), installRootYes, installSudo)
	},
}

func doInstallRoot(version string, yes, sudo bool) error {
	if !confirmInstall("Oakestra root orchestrator", yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := checkFundamentals(); err != nil {
		return err
	}
	setVersion(version)
	fmt.Println("Installing root orchestrator…")
	if err := shellRun(sudo, "curl -sfL oakestra.io/install-root.sh | sh"); err != nil {
		return err
	}

	// Detect and save this machine's IP so the CLI points at the local root orchestrator.
	if ip, err := getLocalIP(); err == nil && ip != "" {
		if err := config.Set("system_manager_ip", ip); err == nil {
			fmt.Printf("%s CLI configured: system_manager_ip = %s\n", green("✓"), bold(ip))
		}
	} else {
		fmt.Printf("%s Could not detect local IP — run: %s\n",
			yellow("Warning:"), bold("oak config set system_manager_ip <IP>"))
	}
	return nil
}

func doInstall1DOC(version string, yes, sudo bool) error {
	if !confirmInstall("Oakestra root and cluster orchestrator", yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := checkFundamentals(); err != nil {
		return err
	}
	setVersion(version)
	fmt.Println("Installing root and cluster orchestrator…")
	if err := shellRun(sudo, "curl -sfL oakestra.io/getstarted.sh | sh"); err != nil {
		return err
	}

	// Detect and save this machine's IP so the CLI points at the local root orchestrator.
	if ip, err := getLocalIP(); err == nil && ip != "" {
		if err := config.Set("system_manager_ip", ip); err == nil {
			fmt.Printf("%s CLI configured: system_manager_ip = %s\n", green("✓"), bold(ip))
		}
	} else {
		fmt.Printf("%s Could not detect local IP — run: %s\n",
			yellow("Warning:"), bold("oak config set system_manager_ip <IP>"))
	}
	return nil
}

// ─── install cluster ──────────────────────────────────────────────────────────

var installClusterCmd = &cobra.Command{
	Use:   "cluster [version]",
	Short: "Install the Oakestra cluster orchestrator",
	Long: `Install the Oakestra cluster orchestrator on this machine.

Requires Docker with Compose support.
Optionally specify a version tag (e.g. v0.4.400).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doInstallCluster(firstArg(args), installClusterYes, installSudo)
	},
}

func doInstallCluster(version string, yes, sudo bool) error {
	if !confirmInstall("Oakestra cluster orchestrator", yes) {
		fmt.Println("Aborted.")
		return nil
	}
	if err := checkFundamentals(); err != nil {
		return err
	}
	setVersion(version)

	// Export SYSTEM_MANAGER_URL if the CLI already knows the root orchestrator IP.
	if cfg, err := config.Load(); err == nil && cfg.SystemManagerIP != "" {
		os.Setenv("SYSTEM_MANAGER_URL", cfg.SystemManagerIP) //nolint:errcheck
		fmt.Printf("Using SYSTEM_MANAGER_URL=%s\n", cfg.SystemManagerIP)
	}

	// Export CLUSTERN_NAME if configured, so the install script can auto-register the cluster.
	if cfg, err := config.Load(); err == nil && cfg.ClusterName != "" {
		os.Setenv("CLUSTER_NAME", cfg.ClusterName) //nolint:errcheck
		fmt.Printf("Using CLUSTER_NAME=%s\n", cfg.ClusterName)
	}

	fmt.Println("Installing cluster orchestrator…")
	return shellRun(sudo, "curl -sfL oakestra.io/install-cluster.sh | sh")
}

// getLocalIP returns the machine's outbound IP address.
// On macOS it reads the primary Wi-Fi/Ethernet interface; on Linux it uses
// the source address that would be used to reach 1.1.1.1.
func getLocalIP() (string, error) {
	var out []byte
	var err error
	if runtime.GOOS == "darwin" {
		out, err = exec.Command("ipconfig", "getifaddr", "en0").Output()
	} else {
		out, err = exec.Command("sh", "-c",
			`ip route get 1.1.1.1 | grep -oP 'src \K\S+'`).Output()
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ─── install worker ───────────────────────────────────────────────────────────

var installWorkerCmd = &cobra.Command{
	Use:   "worker [version]",
	Short: "Install a NodeEngine worker node",
	Long: `Install a NodeEngine worker node on this machine.

Prerequisites:
  - At least one cluster orchestrator must be running and registered
    with the root orchestrator.

After installation the CLI will let you pick which cluster this
worker should join and will configure NodeEngine automatically.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return install.DoInstallWorker(firstArg(args), installWorkerYes)
	},
}

// ─── install full ─────────────────────────────────────────────────────────────

var installFullCmd = &cobra.Command{
	Use:   "full [version]",
	Short: "Install root orchestrator, cluster orchestrator, and worker node",
	Long: `Install the full Oakestra stack on this machine:
  1. Root orchestrator
  2. Cluster orchestrator
  3. NodeEngine worker node

Requires Docker with Compose support.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := firstArg(args)
		yes := installFullYes

		if !confirmInstall("Oakestra root orchestrator, cluster orchestrator, and worker node", yes) {
			fmt.Println("Aborted.")
			return nil
		}
		// Individual confirmations are skipped — user already confirmed above.
		if err := doInstall1DOC(version, true, installSudo); err != nil {
			return fmt.Errorf("root install: %w", err)
		}
		// Wait untill the orchestrators are up and registered before proceeding with the worker install, otherwise it will fail to auto-configure the cluster.
		fmt.Println("Waiting for orchestrators to start…")
		attempt := 0
		client, err := api.New()
		if err != nil {
			return err
		}
		for attempt < 5 {
			clusters, err := client.GetClusters(clusterListAll)
			if err != nil {
				return err
			}
			if len(clusters) == 0 {
				attempt++
				time.Sleep(15 * time.Second)
			} else {
				break
			}
		}
		if attempt == 5 {
			fmt.Println(yellow("Warning: Configuration problem detected. The cluster is not available in the cluster's list. We're aborting the worker installation step to avoid a broken setup. Please check that the cluster orchestrator is running and registered with the root orchestrator, then run 'oak install worker' separately."))
			return nil
		}
		// Worker start prompt still respects the yes flag.
		if err := install.DoInstallWorker(version, yes); err != nil {
			return fmt.Errorf("worker install: %w", err)
		}
		return nil
	},
}

// ─── shared helpers ───────────────────────────────────────────────────────────

// confirmInstall prints a prompt and returns true if the user agrees or yes==true.
func confirmInstall(component string, yes bool) bool {
	if yes {
		return true
	}
	fmt.Printf("This will install the %s on this machine. Continue? [y/N] ", bold(component))
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return ans == "y" || ans == "yes"
}

// checkFundamentals verifies that the prerequisite tools (git, docker, docker compose) are present.
// It prints a status line for each dependency and returns an error listing anything missing.
func checkFundamentals() error {
	type dep struct {
		name    string
		test    func() bool
		install string
	}
	deps := []dep{
		{
			name:    "git",
			test:    func() bool { return exec.Command("git", "--version").Run() == nil },
			install: "https://git-scm.com/downloads",
		},
		{
			name:    "docker",
			test:    func() bool { return exec.Command("docker", "info").Run() == nil },
			install: "https://docs.docker.com/engine/install/",
		},
		{
			name: "docker compose",
			test: func() bool {
				return exec.Command("docker", "compose", "version").Run() == nil ||
					exec.Command("docker-compose", "version").Run() == nil
			},
			install: "https://docs.docker.com/compose/install/",
		},
	}

	fmt.Println(bold("Checking prerequisites…"))
	var missing []string
	for _, d := range deps {
		if d.test() {
			fmt.Printf("  %s %s\n", green("✓"), d.name)
		} else {
			fmt.Printf("  %s %s  — install: %s\n", red("✗"), d.name, dim(d.install))
			missing = append(missing, d.name)
		}
	}
	fmt.Println()

	if len(missing) > 0 {
		return fmt.Errorf("missing prerequisites: %s\nInstall them and try again.",
			strings.Join(missing, ", "))
	}
	return nil
}

// setVersion exports OAKESTRA_VERSION if a non-empty version is given.
func setVersion(version string) {
	if version != "" {
		os.Setenv("OAKESTRA_VERSION", version) //nolint:errcheck
		fmt.Printf("Using OAKESTRA_VERSION=%s\n", version)
	}
}

// shellRun runs a shell command, optionally prefixed with sudo.
func shellRun(sudo bool, command string) error {
	if sudo {
		return runInteractive("sudo", "-E", "sh", "-c", command)
	}
	return runInteractive("sh", "-c", command)
}

// runInteractive runs a command with stdin/stdout/stderr attached to the terminal.
func runInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
