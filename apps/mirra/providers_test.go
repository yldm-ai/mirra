package main

import (
	"strings"
	"testing"

	"github.com/yldm-ai/mirra/apps/mirra/locales"
)

// TestPrivacyLine says where the requests go, which differs by provider: a browser keeps no key, a gateway passes the request on to a service behind it, a server the user runs answers it, and a cloud service is asked directly.
func TestPrivacyLine(t *testing.T) {
	en := locales.In("en")
	for _, c := range []struct {
		name     string
		provider provider
		keyKept  bool
		want     string
	}{
		{"a browser", providerByID(providerOpenAI), false, en.PrivacyWeb()},
		{"a gateway", providerByID("magpie"), true, en.PrivacyGateway()},
		{"a server the user runs", providerByID("sdcpp"), true, en.PrivacyLocal()},
		{"a cloud service", providerByID(providerOpenAI), true, en.PrivacyCloud()},
	} {
		if got := privacyLine(c.provider, c.keyKept, en); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// TestTheGatewayIsNotFreeOfCharge says a gateway is not a server of the user's own: it passes the request on to a service behind it, whose plan pays for it.
func TestTheGatewayIsNotFreeOfCharge(t *testing.T) {
	p := providerByID("magpie")
	if !p.Gateway || p.OwnServer {
		t.Errorf("Magpie = %+v, want a gateway and not a server of the user's own", p)
	}
	gateway := privacyLine(p, true, locales.In("en"))
	if strings.Contains(gateway, locales.In("en").PrivacyLocal()) || gateway == locales.In("en").PrivacyCloud() {
		t.Errorf("a gateway says %q", gateway)
	}
}
