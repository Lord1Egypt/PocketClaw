package main

import "github.com/sipeed/picoclaw/pkg/androiddns"

func init() {
	androiddns.ConfigureDefaultResolverFromEnvironment()
}
