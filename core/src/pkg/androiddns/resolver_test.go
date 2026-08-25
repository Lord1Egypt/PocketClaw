package androiddns

import (
	"reflect"
	"testing"
)

func TestParseServersNormalizesAndroidDNSAddresses(t *testing.T) {
	got := parseServers(" 192.0.2.53 ; [2001:db8::53]:5353 ; 2001:db8::54 ; 192.0.2.53 ")
	want := []string{"192.0.2.53:53", "[2001:db8::53]:5353", "[2001:db8::54]:53"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseServers() = %#v, want %#v", got, want)
	}
}

func TestParseServersIgnoresBlankAddresses(t *testing.T) {
	if got := parseServers(" ; \t ; "); len(got) != 0 {
		t.Fatalf("parseServers() = %#v, want no servers", got)
	}
}
