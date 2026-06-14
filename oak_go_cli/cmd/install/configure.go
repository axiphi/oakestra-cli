package install

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/oakestra/oak-go-cli/internal/api"
	"github.com/oakestra/oak-go-cli/internal/cliout"
	"github.com/oakestra/oak-go-cli/internal/cmd"
	"github.com/oakestra/oak-go-cli/internal/config"
	"github.com/oakestra/oak-go-cli/internal/iotools"
)

func configureWorker() error {
	client, err := api.New()
	if err != nil {
		return err
	}

	clusters, err := client.GetClusters(false)
	if err != nil {
		return err
	}

	if len(clusters) <= 0 {
		return fmt.Errorf("no active clusters found")
	}

	if err := configureWorkerCluster(clusters); err != nil {
		return err
	}

	return nil
}

// `sudo NodeEngine config cluster <IP>` for the selected one.
func configureWorkerCluster(clusters []api.Cluster) error {
	var rootIP string
	if cfg, err := config.Load(); err == nil {
		rootIP = cfg.SystemManagerIP
	}

	cliout.Infof("Probing cluster reachability...")
	clusterProbes := probeAllClusters(clusters, rootIP)
	selectedClusterProbe, err := selectCluster(clusterProbes)
	if err != nil {
		return err
	}

	cliout.Infof("Configuring NodeEngine cluster ip to %s...", cliout.Green(selectedClusterProbe.ip))
	return cmd.RunSilent("sudo", "NodeEngine", "config", "cluster", selectedClusterProbe.ip)
}

func selectCluster(probes []clusterProbe) (*clusterProbe, error) {
	if len(probes) == 1 {
		probe := &probes[0]
		cliout.Infof("Using the only configured cluster: %s (%s)", cliout.Cyan(probe.name), probe.ip)
		return probe, nil
	}

	var options []huh.Option[*clusterProbe]
	for i := range probes {
		probe := &probes[i]

		var statusLabel, ipDisplay string
		if probe.reachability == clusterReachabilityReachable {
			statusLabel = cliout.Green("✓ reachable")
			ipDisplay = cliout.Green(probe.ip)
		} else if probe.reachability == clusterReachabilityReachableViaRoot {
			statusLabel = cliout.Yellow("⚠ reachable via root")
			ipDisplay = cliout.Yellow(probe.ip)
		} else {
			statusLabel = cliout.Red("✗ unreachable")
			ipDisplay = cliout.Dim(probe.ip)
		}

		label := fmt.Sprintf("%s  %s  %s", cliout.Cyan(probe.name), ipDisplay, statusLabel)
		options = append(options, huh.NewOption(label, probe))
	}

	var selected *clusterProbe
	err := huh.NewSelect[*clusterProbe]().
		Title("Select a cluster:").
		Options(options...).
		Value(&selected).
		Run()

	if err != nil {
		return nil, fmt.Errorf("cluster selection failed/aborted: %w", err)
	}

	// TODO: check if this can happen and require selection in case it does
	if selected == nil {
		return nil, fmt.Errorf("no cluster was selected")
	}

	return selected, nil
}

type clusterReachability int

const (
	clusterReachabilityUnreachable clusterReachability = iota
	clusterReachabilityReachable
	clusterReachabilityReachableViaRoot
)

type clusterProbe struct {
	name         string
	reachability clusterReachability
	ip           string
}

// probeAllClusters probes every cluster in parallel.
// For each one it tries CLUSTER_IP first, then ROOT_IP as a fallback.
func probeAllClusters(clusters []api.Cluster, rootIP string) []clusterProbe {
	results := make([]clusterProbe, len(clusters))
	var wg sync.WaitGroup
	wg.Add(len(clusters))
	for i, c := range clusters {
		go func() {
			defer wg.Done()
			pr := clusterProbe{name: c.ClusterName}
			if probeCluster(c, c.ClusterIP) {
				pr.reachability = clusterReachabilityReachable
				pr.ip = c.ClusterIP
			} else if rootIP != "" && rootIP != c.ClusterIP && probeCluster(c, rootIP) {
				pr.reachability = clusterReachabilityReachableViaRoot
				pr.ip = rootIP
			} else {
				pr.reachability = clusterReachabilityUnreachable
				pr.ip = c.ClusterIP
			}
			results[i] = pr
		}()
	}
	wg.Wait()
	return results
}

// probeCluster GETs <ip>:10100/status with a 3 s timeout and returns true
// only if the response parses and cluster_id matches the expected value.
func probeCluster(cluster api.Cluster, ip string) bool {
	type statusResp struct {
		ClusterID string `json:"cluster_id"`
	}
	c := &http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get(fmt.Sprintf("http://%s:10100/api/cluster/status", ip))
	if err != nil {
		return false
	}
	defer iotools.CloseOrWarn(resp.Body, "HTTP response for cluster probe")
	var s statusResp
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return false
	}
	return s.ClusterID == cluster.ClusterID
}
