package v3

import (
	"fmt"
	"strings"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/id"
	"github.com/api7/apisix-mesh-agent/pkg/set"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

const (
	_defaultRoutePriority = 999
)

func (adaptor *adaptor) TranslateRouteConfiguration(r *routev3.RouteConfiguration, opts *TranslateOptions) ([]*apisix.Route, error) {
	var routes []*apisix.Route
	for _, vhost := range r.GetVirtualHosts() {
		partial, err := adaptor.translateVirtualHost(r.Name, vhost, opts)
		if err != nil {
			adaptor.logger.Errorw("failed to translate VirtualHost",
				zap.Error(err),
			)
			return nil, err
		}
		routes = append(routes, partial...)
	}
	if opts != nil && opts.RouteOriginalDestination != nil {
		origDst, ok := opts.RouteOriginalDestination[r.Name]
		if ok {
			patchRoutesWithOriginalDestination(routes, origDst)
		}
	}
	// TODO support Vhds.
	return routes, nil
}

func (adaptor *adaptor) translateVirtualHost(prefix string, vhost *routev3.VirtualHost, opts *TranslateOptions) ([]*apisix.Route, error) {
	if prefix == "" {
		prefix = "<anon>"
	}

	hostSet := set.StringSet{}
	for _, domain := range vhost.Domains {
		if domain == "*" {
			// If this route allows any domain to use, just don't set hosts
			// in APISIX routes.
			hostSet = set.StringSet{}
			break
		} else {
			if pos := strings.Index(domain, ":"); pos != -1 {
				domain = domain[:pos]
			}
			hostSet.Add(domain)
		}
	}
	// avoid unstable array for diff
	hosts := hostSet.OrderedStrings()

	var routes []*apisix.Route
	// FIXME Respect the CaseSensitive field.
	for _, route := range vhost.GetRoutes() {
		sensitive := route.GetMatch().CaseSensitive
		if sensitive != nil && !sensitive.GetValue() {
			// Apache APISIX doesn't support case insensitive URI match,
			// so these routes should be neglected.
			adaptor.logger.Warnw("ignore route with case insensitive match",
				zap.Any("route", route),
			)
			continue
		}

		cluster, skip := adaptor.getClusterName(route)
		if skip {
			continue
		}
		uri, skip := adaptor.getURL(route)
		if skip {
			continue
		}

		name := route.Name
		if name == "" {
			name = "<anon>"
		}
		priority := _defaultRoutePriority
		// This is for istio.
		// use the default and lowest priority for the "allow_any" route.
		if name == "allow_any" {
			priority = 0
		}

		queryVars, skip := adaptor.getParametersMatchVars(route)
		if skip {
			continue
		}
		vars, skip := adaptor.getHeadersMatchVars(route)
		if skip {
			continue
		}
		vars = append(vars, queryVars...)
		name = fmt.Sprintf("%s#%s#%s", name, vhost.GetName(), prefix)
		r := &apisix.Route{
			Name:       name,
			Priority:   int32(priority),
			Status:     1,
			Id:         id.GenID(name),
			Hosts:      hosts,
			Uris:       []string{uri},
			UpstreamId: id.GenID(cluster),
		}
		if len(vars) > 0 {
			r.Vars = vars
		}

		plugin, err := adaptor.translateRouteAction(route)
		if err != nil {
			return nil, err
		}
		if plugin != nil {
			if len(vars) > 0 && plugin.TrafficSplit != nil {
				plugin.TrafficSplit.Rules[0].Match = vars
			}
			r.Plugins = plugin
		}

		routes = append(routes, r)
	}
	adaptor.logger.Debugw("translated apisix routes",
		zap.Any("routes", routes),
	)
	return routes, nil
}

func (adaptor *adaptor) getClusterName(route *routev3.Route) (string, bool) {
	action, ok := route.GetAction().(*routev3.Route_Route)
	if !ok {
		adaptor.logger.Warnw("ignore route with unexpected action",
			zap.Any("route", route),
		)
		return "", true
	}
	switch action.Route.GetClusterSpecifier().(type) {
	case *routev3.RouteAction_Cluster:
		return action.Route.GetClusterSpecifier().(*routev3.RouteAction_Cluster).Cluster, false
	case *routev3.RouteAction_WeightedClusters:
		clusters := action.Route.GetClusterSpecifier().(*routev3.RouteAction_WeightedClusters).WeightedClusters.Clusters
		// pick last cluster as default upstream
		return clusters[len(clusters)-1].Name, false
	default:
		adaptor.logger.Warnw("ignore route with unexpected cluster specifier",
			zap.Any("route", route),
		)
		return "", true
	}
}

func (adaptor *adaptor) getURL(route *routev3.Route) (string, bool) {
	var uri string
	pathSpecifier := route.GetMatch().GetPathSpecifier()
	switch pathSpecifier.(type) {
	case *routev3.RouteMatch_Path:
		uri = pathSpecifier.(*routev3.RouteMatch_Path).Path
	case *routev3.RouteMatch_Prefix:
		uri = pathSpecifier.(*routev3.RouteMatch_Prefix).Prefix + "*"
	default:
		adaptor.logger.Warnw("ignore route with unexpected path specifier",
			zap.Any("route", route),
		)
		return "", true
	}
	return uri, false
}

func (adaptor *adaptor) getParametersMatchVars(route *routev3.Route) ([]*apisix.Var, bool) {
	// See https://github.com/api7/lua-resty-expr
	// for the translation details.
	var vars []*apisix.Var
	for _, param := range route.GetMatch().GetQueryParameters() {
		var expr apisix.Var
		name := "arg_" + param.GetName()
		switch param.GetQueryParameterMatchSpecifier().(type) {
		case *routev3.QueryParameterMatcher_PresentMatch:
			expr.Vars = []string{name, "!", "~~", "^$"}
		case *routev3.QueryParameterMatcher_StringMatch:
			matcher := param.GetQueryParameterMatchSpecifier().(*routev3.QueryParameterMatcher_StringMatch)
			value := getStringMatchValue(matcher.StringMatch)
			op := "~~"
			if matcher.StringMatch.IgnoreCase {
				op = "~*"
			}
			expr.Vars = []string{name, op, value}
		default:
			continue
		}
		vars = append(vars, &expr)
	}
	return vars, false
}

func (adaptor *adaptor) getHeadersMatchVars(route *routev3.Route) ([]*apisix.Var, bool) {
	// See https://github.com/api7/lua-resty-expr
	// for field `vars` syntax.
	var vars []*apisix.Var
	for _, header := range route.GetMatch().GetHeaders() {
		var (
			expr  apisix.Var
			name  string
			value string
		)
		// todo `:scheme`
		switch header.GetName() {
		case ":method": // Istio HeaderMethod
			name = "request_method"
		case ":authority": // Istio HeaderAuthority
			name = "http_host"
		default:
			name = strings.ToLower(header.Name)
			name = "http_" + strings.ReplaceAll(name, "-", "_")
		}

		switch header.HeaderMatchSpecifier.(type) {
		case *routev3.HeaderMatcher_ContainsMatch:
			value = header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_ContainsMatch).ContainsMatch
		case *routev3.HeaderMatcher_ExactMatch:
			value = "^" + header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_ExactMatch).ExactMatch + "$"
		case *routev3.HeaderMatcher_PrefixMatch:
			value = "^" + header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_PrefixMatch).PrefixMatch
		case *routev3.HeaderMatcher_PresentMatch:
		case *routev3.HeaderMatcher_SafeRegexMatch:
			value = header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_SafeRegexMatch).SafeRegexMatch.Regex
		case *routev3.HeaderMatcher_SuffixMatch:
			value = header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_SuffixMatch).SuffixMatch + "$"
		default:
			// TODO Some other HeaderMatchers can be implemented else.
			adaptor.logger.Warnw("ignore route with unexpected header matcher",
				zap.Any("route", route),
			)
			return nil, true
		}

		if header.InvertMatch {
			expr.Vars = []string{name, "!", "~~", value}
		} else {
			expr.Vars = []string{name, "~~", value}
		}
		vars = append(vars, &expr)
	}
	return vars, false
}

func getStringMatchValue(matcher *matcherv3.StringMatcher) string {
	pattern := matcher.MatchPattern
	switch pat := pattern.(type) {
	case *matcherv3.StringMatcher_Exact:
		return "^" + pat.Exact + "$"
	case *matcherv3.StringMatcher_Contains:
		return pat.Contains
	case *matcherv3.StringMatcher_Prefix:
		return "^" + pat.Prefix
	case *matcherv3.StringMatcher_Suffix:
		return pat.Suffix + "$"
	case *matcherv3.StringMatcher_SafeRegex:
		// TODO Regex Engine detection.
		return pat.SafeRegex.Regex
	default:
		panic("unknown StringMatcher type")
	}
}

func patchRoutesWithOriginalDestination(routes []*apisix.Route, origDst string) {
	if strings.HasPrefix(origDst, "0.0.0.0:") {
		port := origDst[len("0.0.0.0:"):]
		for _, r := range routes {
			r.Vars = append(r.Vars, &apisix.Var{
				Vars: []string{"connection_original_dst", "~~", port + "$"},
			})
		}
	} else {
		for _, r := range routes {
			r.Vars = append(r.Vars, &apisix.Var{
				Vars: []string{"connection_original_dst", "==", origDst},
			})
		}
	}
}

// translateRouteAction translates envoy RouteAction to traffic-split plugins configs
func (adaptor *adaptor) translateRouteAction(r *routev3.Route) (*apisix.Plugins, error) {
	switch r.GetAction().(type) {
	case *routev3.Route_Route:
	default:
		adaptor.logger.Infow("ignore unsupported route action",
			zap.Any("action", r.GetAction()),
		)
		return nil, nil
	}
	action := r.GetAction().(*routev3.Route_Route).Route
	switch action.GetClusterSpecifier().(type) {
	case *routev3.RouteAction_WeightedClusters:
	default:
		adaptor.logger.Debugw("ignore single cluster",
			zap.Any("cluster_specifier", action.GetClusterSpecifier()),
		)
		return nil, nil
	}
	clusters := action.GetClusterSpecifier().(*routev3.RouteAction_WeightedClusters).WeightedClusters.Clusters

	weighted := make([]*apisix.TrafficSplitWeightedUpstreams, len(clusters))
	for i, weightedCluster := range clusters {
		adaptor.logger.Debugw("translating weighted cluster",
			zap.Any("cluster", weightedCluster),
			zap.Any("index", i),
		)
		weighted[i] = &apisix.TrafficSplitWeightedUpstreams{
			Weight: weightedCluster.Weight.Value,
		}
		if i != len(clusters)-1 {
			// last one is default upstream
			weighted[i].UpstreamId = id.GenID(weightedCluster.Name)
		}
	}

	return &apisix.Plugins{
		TrafficSplit: &apisix.TrafficSplit{
			Rules: []*apisix.TrafficSplitRule{
				{
					WeightedUpstreams: weighted,
				},
			},
		},
	}, nil
}
