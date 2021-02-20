package v3

import (
	"fmt"
	"strings"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"go.uber.org/zap"

	"github.com/api7/apisix-mesh-agent/pkg/id"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
)

func (adaptor *xdsAdaptor) TranslateRouteConfiguration(r *routev3.RouteConfiguration) ([]*apisix.Route, error) {
	var routes []*apisix.Route
	for _, vhost := range r.GetVirtualHosts() {
		partial, err := adaptor.translateVirtualHost(r.Name, vhost)
		if err != nil {
			adaptor.logger.Errorw("failed to translate VirtualHost",
				zap.Error(err),
			)
			return nil, err
		}
		routes = append(routes, partial...)
	}
	// TODO support Vhds.
	return routes, nil
}

func (adaptor *xdsAdaptor) translateVirtualHost(prefix string, vhost *routev3.VirtualHost) ([]*apisix.Route, error) {
	if prefix == "" {
		prefix = "<anon>"
	}
	var routes []*apisix.Route

	// FIXME Respect the CaseSensitive field.
	for _, route := range vhost.GetRoutes() {
		if route.GetMatch().CaseSensitive.GetValue() == false {
			// Apache APISIX doens't support case insensitive URI match,
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

		queryVars, skip := adaptor.getParametersMatchVars(route)
		if skip {
			continue
		}
		vars, skip := adaptor.getHeadersMatchVars(route)
		if skip {
			continue
		}
		vars = append(vars, queryVars...)
		name = fmt.Sprintf("%s#%s#%s", prefix, vhost.GetName(), name)
		r := &apisix.Route{
			Name:   name,
			Status: 1,
			Id: &apisix.ID{
				OneofId: &apisix.ID_StrVal{
					StrVal: id.GenID(name),
				},
			},
			Hosts: vhost.Domains,
			Uris:  []string{uri},
			UpstreamId: &apisix.ID{
				OneofId: &apisix.ID_StrVal{
					StrVal: id.GenID(cluster),
				},
			},
			Vars: vars,
		}
		routes = append(routes, r)
	}
	return routes, nil
}

func (adaptor *xdsAdaptor) getClusterName(route *routev3.Route) (string, bool) {
	action, ok := route.GetAction().(*routev3.Route_Route)
	if !ok {
		adaptor.logger.Warnw("ignore route with unexpected action",
			zap.Any("route", route),
		)
		return "", true
	}
	cluster, ok := action.Route.GetClusterSpecifier().(*routev3.RouteAction_Cluster)
	if !ok {
		adaptor.logger.Warnw("ignore route with unexpected cluster specifier",
			zap.Any("route", route),
		)
		return "", true
	}
	return cluster.Cluster, false
}

func (adaptor *xdsAdaptor) getURL(route *routev3.Route) (string, bool) {
	var uri string
	switch route.GetMatch().GetPathSpecifier().(type) {
	case *routev3.RouteMatch_Path:
		uri = route.GetMatch().GetPathSpecifier().(*routev3.RouteMatch_Path).Path
	case *routev3.RouteMatch_Prefix:
		uri = route.GetMatch().GetPathSpecifier().(*routev3.RouteMatch_Prefix).Prefix + "*"
	default:
		adaptor.logger.Warnw("ignore route with unexpected path specifier",
			zap.Any("route", route),
		)
		return "", true
	}
	return uri, false
}

func (adaptor *xdsAdaptor) getParametersMatchVars(route *routev3.Route) ([]*apisix.Var, bool) {
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
		}
		vars = append(vars, &expr)
	}
	return vars, false
}

func (adaptor *xdsAdaptor) getHeadersMatchVars(route *routev3.Route) ([]*apisix.Var, bool) {
	// See https://github.com/api7/lua-resty-expr
	// for the translation details.
	var vars []*apisix.Var
	for _, header := range route.GetMatch().GetHeaders() {
		var (
			expr  apisix.Var
			name  string
			value string
		)
		switch header.GetName() {
		case ":method":
			name = "request_method"
		case ":authority":
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
	switch pattern.(type) {
	case *matcherv3.StringMatcher_Exact:
		return "^" + pattern.(*matcherv3.StringMatcher_Exact).Exact + "$"
	case *matcherv3.StringMatcher_Contains:
		return pattern.(*matcherv3.StringMatcher_Contains).Contains
	case *matcherv3.StringMatcher_Prefix:
		return "^" + pattern.(*matcherv3.StringMatcher_Prefix).Prefix
	case *matcherv3.StringMatcher_Suffix:
		return pattern.(*matcherv3.StringMatcher_Suffix).Suffix + "$"
	case *matcherv3.StringMatcher_SafeRegex:
		// TODO Regex Engine detection.
		return pattern.(*matcherv3.StringMatcher_SafeRegex).SafeRegex.Regex
	default:
		panic("unknown StringMatcher type")
	}
}