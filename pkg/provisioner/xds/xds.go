package xds

var (
	_interestedResources = map[string]struct{}{
		"@type.googleapis.com/envoy.api.v2.Route":   {},
		"@type.googleapis.com/envoy.api.v2.Cluster": {},
	}
)

func ResourceInUse(typeUrl string) (ok bool) {
	_, ok = _interestedResources[typeUrl]
	return
}
