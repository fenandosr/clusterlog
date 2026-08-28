package clusterlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Fixtures sintéticos: nunca datos reales de ningún deployment (misma regla
// que ADR-0005 aplica al motor). Los nombres de host, dominios e IPs de
// abajo no corresponden a ninguna infraestructura real.

const testDeploymentYAML = `
version: 1
engine: dev
env: staging
mgmt_cidr: 203.0.113.0/32
aws:
  region: us-east-1
dns:
  provider: route53
  root_domain: example.test
identity:
  keycloak_domain: auth.example.test
  headscale_domain: vpn.example.test
hosts:
  edge:
    provider: aws
    instance_type: t4g.small
    stacks: [base, identity]
  data:
    provider: aws
    instance_type: t4g.small
    stacks: [database]
`

func writeMksrvFixture(t *testing.T, dir string, outputs map[string]any) {
	t.Helper()
	mustWriteTestFile(t, filepath.Join(dir, "deployment.yaml"), testDeploymentYAML)
	data, err := json.Marshal(outputs)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteTestFile(t, filepath.Join(dir, ".mksrv", "infra", "outputs.json"), string(data))
}

func fixtureOutputs() map[string]any {
	return map[string]any{
		"hosts": map[string]any{
			"edge": map[string]any{
				"provider": "aws", "management_ip": "198.51.100.10", "private_ip": "10.0.0.10",
				"public_ip": "198.51.100.10", "instance_id": "i-edge000000000001", "az": "us-east-1a",
			},
			"data": map[string]any{
				"provider": "aws", "management_ip": "198.51.100.11", "private_ip": "10.0.0.11",
				"public_ip": "198.51.100.11", "instance_id": "i-data000000000001", "az": "us-east-1a",
			},
		},
		"network": map[string]any{"vpc_id": "vpc-test", "subnet_id": "subnet-test", "availability_zone": "us-east-1a"},
		"dns":     map[string]any{"created": []string{"auth.example.test"}, "pending": []string{"vpn.example.test"}},
	}
}

func TestImportFromMksrvMapsRolesAndAddresses(t *testing.T) {
	dir := t.TempDir()
	writeMksrvFixture(t, dir, fixtureOutputs())

	topo, warnings, err := importFromMksrv(dir, false)
	if err != nil {
		t.Fatalf("importFromMksrv falló: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("no se esperaban advertencias: %v", warnings)
	}
	if len(topo.Hosts) != 2 {
		t.Fatalf("se esperaban 2 hosts, se obtuvieron %d", len(topo.Hosts))
	}

	byName := map[string]TopologyHost{}
	for _, h := range topo.Hosts {
		byName[h.Name] = h
	}

	edge := byName["edge"]
	if edge.Role != "edge" {
		t.Fatalf("host con stack 'base' debe ser role=edge, fue %q", edge.Role)
	}
	if edge.Addresses.Public != "" {
		t.Fatal("sin --include-public-ip no debe llevar IP pública")
	}
	if edge.Addresses.Private != "10.0.0.10" || edge.Addresses.Management != "198.51.100.10" {
		t.Fatalf("direcciones de edge mal mapeadas: %+v", edge.Addresses)
	}
	if edge.InstanceID != "i-edge000000000001" || edge.AZ != "us-east-1a" {
		t.Fatalf("instance_id/az de edge mal mapeados: %+v", edge)
	}

	data := byName["data"]
	if data.Role != "data" {
		t.Fatalf("host sin stack 'base' debe ser role=data, fue %q", data.Role)
	}

	if topo.Network == nil || topo.Network.VPCID != "vpc-test" {
		t.Fatalf("network mal mapeada: %+v", topo.Network)
	}
	if topo.DNS == nil || topo.DNS.RootDomain != "example.test" {
		t.Fatalf("dns.root_domain debe venir de deployment.yaml, no de outputs.json: %+v", topo.DNS)
	}
}

func TestImportFromMksrvIncludesPublicIPOnlyWhenAsked(t *testing.T) {
	dir := t.TempDir()
	writeMksrvFixture(t, dir, fixtureOutputs())

	topo, _, err := importFromMksrv(dir, true)
	if err != nil {
		t.Fatalf("importFromMksrv falló: %v", err)
	}
	for _, h := range topo.Hosts {
		if h.Addresses.Public == "" {
			t.Fatalf("--include-public-ip debía incluir la IP pública de %s", h.Name)
		}
	}
}

func TestImportFromMksrvWarnsOnMismatchedHosts(t *testing.T) {
	dir := t.TempDir()
	outputs := fixtureOutputs()
	// "data" está en deployment.yaml pero no en outputs.json (falta apply).
	hosts := outputs["hosts"].(map[string]any)
	delete(hosts, "data")
	// "extra" está en outputs.json pero no en deployment.yaml (huérfano).
	hosts["extra"] = map[string]any{"provider": "aws", "management_ip": "198.51.100.99"}
	writeMksrvFixture(t, dir, outputs)

	topo, warnings, err := importFromMksrv(dir, false)
	if err != nil {
		t.Fatalf("importFromMksrv falló: %v", err)
	}
	if len(warnings) != 2 {
		t.Fatalf("se esperaban 2 advertencias (falta apply + host huérfano), se obtuvieron: %v", warnings)
	}
	if len(topo.Hosts) != 2 {
		t.Fatalf("el host huérfano de outputs.json no debe agregarse; se esperaban 2 hosts (edge, data), se obtuvieron %d", len(topo.Hosts))
	}
	for _, h := range topo.Hosts {
		if h.Name == "data" && h.Addresses.Private != "" {
			t.Fatal("data sin outputs no debería tener direcciones")
		}
		if h.Name == "extra" {
			t.Fatal("un host huérfano de outputs.json no declarado en deployment.yaml no debe aparecer en topology.Hosts")
		}
	}
}

func TestImportFromMksrvMissingWorkspace(t *testing.T) {
	_, _, err := importFromMksrv(filepath.Join(t.TempDir(), "no-existe"), false)
	if err == nil {
		t.Fatal("se esperaba un error por workspace inexistente")
	}
	cliErr, ok := err.(*CLIError)
	if !ok || cliErr.Kind != "topology_source_missing" {
		t.Fatalf("se esperaba topology_source_missing, se obtuvo: %v", err)
	}
}

func TestRunTopologyImportWritesFile(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "instancia")
	mustWriteTestFile(t, filepath.Join(root, "zola.toml"), "title = \"x\"\n")
	workspace := filepath.Join(base, "mksrv-workspace")
	writeMksrvFixture(t, workspace, fixtureOutputs())

	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	if err := app.runTopology(root, []string{"import", "--from-mksrv", workspace}); err != nil {
		t.Fatalf("topology import falló: %v", err)
	}

	var topo Topology
	if err := loadJSON(filepath.Join(root, "data", "topology.json"), &topo); err != nil {
		t.Fatalf("no se pudo leer data/topology.json: %v", err)
	}
	if len(topo.Hosts) != 2 {
		t.Fatalf("se esperaban 2 hosts en el archivo escrito, se obtuvieron %d", len(topo.Hosts))
	}
}

func TestRunTopologyImportDryRunDoesNotWrite(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "instancia")
	mustWriteTestFile(t, filepath.Join(root, "zola.toml"), "title = \"x\"\n")
	workspace := filepath.Join(base, "mksrv-workspace")
	writeMksrvFixture(t, workspace, fixtureOutputs())

	app := New(Options{JSON: true, DryRun: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	if err := app.runTopology(root, []string{"import", "--from-mksrv", workspace}); err != nil {
		t.Fatalf("topology import --dry-run falló: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "data", "topology.json")); err == nil {
		t.Fatal("--dry-run no debería escribir data/topology.json")
	}
}

func TestTopologyMissingFromMksrvFlag(t *testing.T) {
	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	err := app.runTopology(t.TempDir(), []string{"import"})
	if err == nil {
		t.Fatal("se esperaba un error por --from-mksrv faltante")
	}
	cliErr, ok := err.(*CLIError)
	if !ok || cliErr.Kind != "missing_source" {
		t.Fatalf("se esperaba missing_source, se obtuvo: %v", err)
	}
}

func TestTopologyUnknownSubcommand(t *testing.T) {
	app := New(Options{JSON: true, Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin, Executable: "clusterlog"})
	err := app.runTopology(t.TempDir(), []string{"bogus"})
	if err == nil {
		t.Fatal("se esperaba un error por subcomando desconocido")
	}
	cliErr, ok := err.(*CLIError)
	if !ok || cliErr.Kind != "unknown_subcommand" {
		t.Fatalf("se esperaba unknown_subcommand, se obtuvo: %v", err)
	}
}
