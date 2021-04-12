package controlplane

// ControlPlane represents the control plane in e2e test cases.
type ControlPlane interface {
	// Type returns the control plane type.
	Type() string
	// Namespace fetches the deployed namespace of control plane components.
	Namespace() string
	// Deploy deploys the control plane.
	Deploy() error
	// Uninstall uninstalls the control plane.
	Uninstall() error
	// Addr returns the address to communicate with the control plane for fetching
	// configuration changes.
	Addr() string
}
