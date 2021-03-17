package controlplane

type istio struct {
	namespace string
}

type IstioOptions struct {
	InjectorTemplate string
}

// NewIstioControlPlane creates an istio control plane.
func NewIstioControlPlane() ControlPlane {
	return &istio{}
}

func (cp *istio) Namespace() string {
	return cp.namespace
}

func (cp *istio) Type() string {
	return "istio"
}
