package clusterlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Este archivo implementa el único adaptador de topología soportado hoy:
// mksrv (proyecto hermano de clusterlog, mismo autor, mismo patrón
// motor/instancia). El contrato de salida (Topology) es genérico a
// propósito -- no expone tipos de mksrv fuera de este archivo -- para que un
// adaptador futuro de otra herramienta pueda producir la misma forma sin
// tocar nada río abajo (data/topology.json, y eventualmente la sección Zola
// que lo consuma).
//
// Prototipo: valida el mapeo contra un workspace mksrv real antes de
// formalizarlo en un ADR. Ver docs/adr/0007-topologia-importada.md.

// Topology es el contrato genérico que cualquier adaptador debe producir.
type Topology struct {
	Version     int              `json:"version"`
	Source      string           `json:"source"`
	GeneratedBy string           `json:"generated_by"`
	Env         string           `json:"env,omitempty"`
	Hosts       []TopologyHost   `json:"hosts"`
	Network     *TopologyNetwork `json:"network,omitempty"`
	DNS         *TopologyDNS     `json:"dns,omitempty"`
}

type TopologyHost struct {
	Name         string            `json:"name"`
	Role         string            `json:"role,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	InstanceType string            `json:"instance_type,omitempty"`
	InstanceID   string            `json:"instance_id,omitempty"`
	AZ           string            `json:"az,omitempty"`
	Stacks       []string          `json:"stacks,omitempty"`
	Addresses    TopologyAddresses `json:"addresses"`
}

type TopologyAddresses struct {
	Private    string `json:"private,omitempty"`
	Management string `json:"management,omitempty"`
	Public     string `json:"public,omitempty"`
	Mesh       string `json:"mesh,omitempty"`
}

type TopologyNetwork struct {
	VPCID            string `json:"vpc_id,omitempty"`
	SubnetID         string `json:"subnet_id,omitempty"`
	AvailabilityZone string `json:"availability_zone,omitempty"`
}

type TopologyDNS struct {
	RootDomain string   `json:"root_domain,omitempty"`
	Created    []string `json:"created,omitempty"`
	Pending    []string `json:"pending,omitempty"`
}

// --- Formas de entrada de mksrv (subconjunto mínimo que se usa) ---
//
// Se decodifican sólo los campos que este adaptador necesita, no el
// contrato completo de mksrv (internal/model/model.go allá): si mksrv
// agrega campos nuevos a deployment.yaml u outputs.json, este adaptador los
// ignora en vez de fallar.

type mksrvDeployment struct {
	Env string `yaml:"env"`
	DNS struct {
		RootDomain string `yaml:"root_domain"`
	} `yaml:"dns"`
	Hosts map[string]struct {
		Provider     string   `yaml:"provider"`
		InstanceType string   `yaml:"instance_type"`
		Stacks       []string `yaml:"stacks"`
	} `yaml:"hosts"`
}

type mksrvOutputs struct {
	Hosts map[string]struct {
		Provider     string `json:"provider"`
		ManagementIP string `json:"management_ip"`
		PrivateIP    string `json:"private_ip"`
		PublicIP     string `json:"public_ip"`
		InstanceID   string `json:"instance_id"`
		AZ           string `json:"az"`
	} `json:"hosts"`
	Network struct {
		VPCID            string `json:"vpc_id"`
		SubnetID         string `json:"subnet_id"`
		AvailabilityZone string `json:"availability_zone"`
	} `json:"network"`
	DNS struct {
		Created []string `json:"created"`
		Pending []string `json:"pending"`
	} `json:"dns"`
}

// importFromMksrv lee deployment.yaml y .mksrv/infra/outputs.json (y, si
// existe, .mksrv/mesh.json) de un workspace mksrv y produce el contrato
// genérico Topology. No ejecuta terraform ni mksrv, no hace red: sólo lee
// los archivos que esas herramientas ya dejaron en disco.
func importFromMksrv(workspaceDir string, includePublicIP bool) (*Topology, []string, error) {
	var warnings []string

	depPath := filepath.Join(workspaceDir, "deployment.yaml")
	depData, err := os.ReadFile(depPath)
	if err != nil {
		return nil, nil, NewError(ExitUsage, "topology_source_missing", "no se pudo leer deployment.yaml del workspace mksrv", map[string]any{"path": depPath, "error": err.Error()})
	}
	var dep mksrvDeployment
	if err := yaml.Unmarshal(depData, &dep); err != nil {
		return nil, nil, NewError(ExitUsage, "topology_source_invalid", "deployment.yaml no es YAML válido", map[string]any{"path": depPath, "error": err.Error()})
	}

	outPath := filepath.Join(workspaceDir, ".mksrv", "infra", "outputs.json")
	var out mksrvOutputs
	if err := loadJSON(outPath, &out); err != nil {
		return nil, nil, NewError(ExitUsage, "topology_source_missing", "no se pudo leer .mksrv/infra/outputs.json; corra 'mksrv apply --infra-only' primero", map[string]any{"path": outPath, "error": err.Error()})
	}

	meshIPs := map[string]string{}
	meshPath := filepath.Join(workspaceDir, ".mksrv", "mesh.json")
	if meshData, err := os.ReadFile(meshPath); err == nil {
		if jsonErr := json.Unmarshal(meshData, &meshIPs); jsonErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s no es JSON válido, se ignoran las IP de malla: %s", meshPath, jsonErr.Error()))
		}
	}

	names := make([]string, 0, len(dep.Hosts))
	for name := range dep.Hosts {
		names = append(names, name)
	}
	for name := range out.Hosts {
		if _, ok := dep.Hosts[name]; !ok {
			warnings = append(warnings, fmt.Sprintf("outputs.json tiene el host %q pero deployment.yaml no lo declara; se omite", name))
		}
	}
	sort.Strings(names)

	hosts := make([]TopologyHost, 0, len(names))
	for _, name := range names {
		declared := dep.Hosts[name]
		role := "data"
		for _, s := range declared.Stacks {
			if s == "base" {
				role = "edge"
				break
			}
		}
		h := TopologyHost{
			Name:         name,
			Role:         role,
			Provider:     declared.Provider,
			InstanceType: declared.InstanceType,
			Stacks:       declared.Stacks,
		}
		infraOut, ok := out.Hosts[name]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("%q está en deployment.yaml pero no en outputs.json; corra 'mksrv apply --infra-only'", name))
			hosts = append(hosts, h)
			continue
		}
		h.InstanceID = infraOut.InstanceID
		h.AZ = infraOut.AZ
		h.Addresses = TopologyAddresses{
			Private:    infraOut.PrivateIP,
			Management: infraOut.ManagementIP,
			Mesh:       meshIPs[name],
		}
		if includePublicIP {
			h.Addresses.Public = infraOut.PublicIP
		}
		hosts = append(hosts, h)
	}

	topo := &Topology{
		Version:     1,
		Source:      "mksrv",
		GeneratedBy: "clusterlog topology import --from-mksrv",
		Env:         dep.Env,
		Hosts:       hosts,
		Network: &TopologyNetwork{
			VPCID:            out.Network.VPCID,
			SubnetID:         out.Network.SubnetID,
			AvailabilityZone: out.Network.AvailabilityZone,
		},
		DNS: &TopologyDNS{
			RootDomain: dep.DNS.RootDomain,
			Created:    out.DNS.Created,
			Pending:    out.DNS.Pending,
		},
	}
	return topo, warnings, nil
}
