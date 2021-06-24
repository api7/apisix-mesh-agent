package v3

import (
	"sort"
	"testing"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"

	apisixutil "github.com/api7/apisix-mesh-agent/pkg/apisix"
	"github.com/api7/apisix-mesh-agent/pkg/id"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/types/apisix"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
)

func TestGetStringMatchValue(t *testing.T) {
	matcher := &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_Exact{
			Exact: "Hangzhou",
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), "^Hangzhou$", "translating exact string match")

	matcher = &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_Contains{
			Contains: "Hangzhou",
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), "Hangzhou", "translating exact string match")

	matcher = &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_Prefix{
			Prefix: "Hangzhou",
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), "^Hangzhou", "translating exact string match")

	matcher = &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_Suffix{
			Suffix: "Hangzhou",
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), "Hangzhou$", "translating exact string match")

	matcher = &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_SafeRegex{
			SafeRegex: &matcherv3.RegexMatcher{
				Regex: ".*\\d+Hangzhou",
			},
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), ".*\\d+Hangzhou", "translating exact string match")
}

func TestGetHeadersMatchVars(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}

	route := &routev3.Route{
		Match: &routev3.RouteMatch{
			Headers: []*routev3.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
						ContainsMatch: "POST",
					},
				},
				{
					Name: ":authority",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{
						ExactMatch: "apisix.apache.org",
					},
				},
				{
					Name: "Accept-Ranges",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_PrefixMatch{
						PrefixMatch: "bytes",
					},
					InvertMatch: true,
				},
				{
					Name: "Content-Type",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_SafeRegexMatch{
						SafeRegexMatch: &matcherv3.RegexMatcher{
							Regex: `\.(jpg|png|gif)`,
						},
					},
				},
				{
					Name: "Content-Encoding",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_SuffixMatch{
						SuffixMatch: "zip",
					},
				},
			},
		},
	}
	vars, skip := a.getHeadersMatchVars(route)
	assert.Equal(t, skip, false)
	assert.Len(t, vars, len(route.Match.Headers))
	assert.Equal(t, vars[0], &apisix.Var{
		Vars: []string{"request_method", "~~", "POST"},
	})
	assert.Equal(t, vars[1], &apisix.Var{
		Vars: []string{"http_host", "~~", "^apisix.apache.org$"},
	})
	assert.Equal(t, vars[2], &apisix.Var{
		Vars: []string{"http_accept_ranges", "!", "~~", "^bytes"},
	})
	assert.Equal(t, vars[3], &apisix.Var{
		Vars: []string{"http_content_type", "~~", `\.(jpg|png|gif)`},
	})
	assert.Equal(t, vars[4], &apisix.Var{
		Vars: []string{"http_content_encoding", "~~", "zip$"},
	})
}

func TestGetParametersMatchVars(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}

	route := &routev3.Route{
		Match: &routev3.RouteMatch{
			QueryParameters: []*routev3.QueryParameterMatcher{
				{
					Name: "man",
					QueryParameterMatchSpecifier: &routev3.QueryParameterMatcher_PresentMatch{
						PresentMatch: true,
					},
				},
				{
					Name: "id",
					QueryParameterMatchSpecifier: &routev3.QueryParameterMatcher_StringMatch{
						StringMatch: &matcherv3.StringMatcher{
							MatchPattern: &matcherv3.StringMatcher_Exact{
								Exact: "123456",
							},
						},
					},
				},
				{
					Name: "name",
					QueryParameterMatchSpecifier: &routev3.QueryParameterMatcher_StringMatch{
						StringMatch: &matcherv3.StringMatcher{
							MatchPattern: &matcherv3.StringMatcher_Contains{
								Contains: "alex",
							},
							IgnoreCase: true,
						},
					},
				},
			},
		},
	}

	vars, skip := a.getParametersMatchVars(route)
	assert.Equal(t, skip, false)
	assert.Len(t, vars, 3)
	assert.Equal(t, vars[0], &apisix.Var{
		Vars: []string{"arg_man", "!", "~~", "^$"},
	})
	assert.Equal(t, vars[1], &apisix.Var{
		Vars: []string{"arg_id", "~~", "^123456$"},
	})
	assert.Equal(t, vars[2], &apisix.Var{
		Vars: []string{"arg_name", "~*", "alex"},
	})
}

func TestGetURL(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}
	route := &routev3.Route{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Prefix{
				Prefix: "/foo/baz",
			},
		},
	}
	uri, skip := a.getURL(route)
	assert.Equal(t, skip, false)
	assert.Equal(t, uri, "/foo/baz*")

	route = &routev3.Route{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Path{
				Path: "/foo/baz",
			},
		},
	}
	uri, skip = a.getURL(route)
	assert.Equal(t, skip, false)
	assert.Equal(t, uri, "/foo/baz")

	route = &routev3.Route{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_SafeRegex{
				SafeRegex: &matcherv3.RegexMatcher{
					Regex: "/foo/.*?",
				},
			},
		},
	}
	_, skip = a.getURL(route)
	assert.Equal(t, skip, true)
}

func TestGetClusterName(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}
	route := &routev3.Route{
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_Cluster{
					Cluster: "kubernetes.default.svc.cluster.local",
				},
			},
		},
	}
	clusterName, skip := a.getClusterName(route)
	assert.Equal(t, skip, false)
	assert.Equal(t, clusterName, "kubernetes.default.svc.cluster.local")

	route = &routev3.Route{
		Action: &routev3.Route_Redirect{},
	}
	_, skip = a.getClusterName(route)
	assert.Equal(t, skip, true)

	route = &routev3.Route{
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_ClusterHeader{},
			},
		},
	}
	_, skip = a.getClusterName(route)
	assert.Equal(t, skip, true)
}

func TestTranslateVirtualHost(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}
	vhost := &routev3.VirtualHost{
		Name: "test",
		Domains: []string{
			"apisix.apache.org",
			"*.apache.org",
		},
		Routes: []*routev3.Route{
			{
				Name: "route1",
				Match: &routev3.RouteMatch{
					CaseSensitive: &wrappers.BoolValue{
						Value: true,
					},
					Headers: []*routev3.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
								ContainsMatch: "POST",
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Prefix{
						Prefix: "/foo/baz",
					},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "kubernetes.default.svc.cluster.local",
						},
					},
				},
			},
			{
				Name: "route2",
				Match: &routev3.RouteMatch{
					Headers: []*routev3.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
								ContainsMatch: "POST",
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Path{
						Path: "/foo/baz",
					},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_ClusterHeader{},
					},
				},
			},
			{
				Name: "route3",
				Match: &routev3.RouteMatch{
					CaseSensitive: &wrappers.BoolValue{
						Value: false,
					},
				},
			},
		},
	}
	routes, err := a.translateVirtualHost("test", vhost, nil)
	assert.Nil(t, err)
	assert.Len(t, routes, 1)
	assert.Equal(t, routes[0].Name, "route1#test#test")
	assert.Equal(t, routes[0].Status, apisix.Route_Enable)
	assert.Equal(t, routes[0].Id, id.GenID(routes[0].Name))

	sort.Strings(routes[0].Hosts)
	assert.Equal(t, routes[0].Hosts, []string{
		"*.apache.org",
		"apisix.apache.org",
	})
	assert.Equal(t, routes[0].Uris, []string{
		"/foo/baz*",
	})
	assert.Equal(t, routes[0].UpstreamId, id.GenID("kubernetes.default.svc.cluster.local"))
	assert.Equal(t, routes[0].Vars, []*apisix.Var{
		{
			Vars: []string{"request_method", "~~", "POST"},
		},
	})
}

func TestPatchRoutesWithOriginalDestination(t *testing.T) {
	routes := []*apisix.Route{
		{
			Name: "1",
			Id:   "1",
		},
	}
	patchRoutesWithOriginalDestination(routes, "10.0.5.4:8080")
	assert.Equal(t, routes[0].Vars, []*apisix.Var{
		{
			Vars: []string{
				"connection_original_dst",
				"==",
				"10.0.5.4:8080",
			},
		},
	})
}

func TestUnstableHostsRouteDiff(t *testing.T) {
	a := &adaptor{logger: log.DefaultLogger}
	vhost1 := &routev3.VirtualHost{
		Name: "test",
		Domains: []string{
			"a.apisix.apache.org",
			"b.apisix.apache.org",
			"c.apisix.apache.org",
		},
		Routes: []*routev3.Route{
			{
				Name: "route",
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "kubernetes.default.svc.cluster.local",
						},
					},
				},
				Match: &routev3.RouteMatch{
					Headers: []*routev3.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
								ContainsMatch: "POST",
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Path{
						Path: "/foo/baz",
					},
				},
			},
		},
	}
	vhost2 := &routev3.VirtualHost{
		Name: "test",
		Domains: []string{
			"c.apisix.apache.org",
			"a.apisix.apache.org",
			"b.apisix.apache.org",
		},
		Routes: []*routev3.Route{
			{
				Name: "route",
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "kubernetes.default.svc.cluster.local",
						},
					},
				},
				Match: &routev3.RouteMatch{
					Headers: []*routev3.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
								ContainsMatch: "POST",
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Path{
						Path: "/foo/baz",
					},
				},
			},
		},
	}
	routes1, err := a.translateVirtualHost("test", vhost1, nil)
	assert.Nil(t, err)
	routes2, err := a.translateVirtualHost("test", vhost2, nil)
	assert.Nil(t, err)

	assert.NotNil(t, routes1)
	assert.NotNil(t, routes2)

	added, deleted, updated := apisixutil.CompareRoutes(routes1, routes2)
	assert.Nil(t, added)
	assert.Nil(t, deleted)
	assert.Nil(t, updated)
}
