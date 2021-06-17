package constant

import "os"

var (
	// Helm indicates helm binary filename
	Helm              = "helm"
	// DefaultKubeconfig is default kubeconfig location
	DefaultKubeconfig = "~/.kube/config"
	// E2eHome get e2e directory from environment variable
	E2eHome           = os.Getenv("APISIX_MESH_AGENT_E2E_HOME")
)
