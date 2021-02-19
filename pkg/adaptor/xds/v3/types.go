package xds

// XDSAdaptor translates xDS resources like Route, Cluster
// to the equivalent configs in Apache APISIX.
type XDSAdaptor interface {
	TranslateRouteConfiguration()
}

type xdsAdaptor struct {
}

func
