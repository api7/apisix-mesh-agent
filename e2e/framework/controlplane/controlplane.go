package controlplane

// ControlPlane represents the control plane in e2e test cases.
type ControlPlane interface {
	// Type returns the control plane type.
	Type() string
	// Namespace fetches the deployed namespace of control plane components.
	Namespace() string
}
