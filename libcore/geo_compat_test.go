package libcore

import (
	"os"
	"path/filepath"
	"testing"
)

// Run with the exact databases bundled in the upstream 1.4.2 APK. The optional
// path keeps ordinary unit tests independent of downloaded binary assets.
func TestBundledGeoRulesRemainReadable(t *testing.T) {
	assets:=os.Getenv("NEKO_TEST_ASSETS")
	if assets=="" {t.Skip("set NEKO_TEST_ASSETS to unpacked pinned GeoIP/Geosite databases")}
	g:=new(geoip)
	if err:=g.Open(filepath.Join(assets,"geoip.db"));err!=nil{t.Fatal(err)}
	defer g.geoipReader.Close()
	ipRules,err:=g.Rules("cn")
	if err!=nil{t.Fatal(err)}
	if len(ipRules)==0 || len(ipRules[0].DefaultOptions.IPCIDR)==0{t.Fatal("empty GeoIP rules")}
	s:=new(geosite)
	if err:=s.Open(filepath.Join(assets,"geosite.db"));err!=nil{t.Fatal(err)}
	siteRules,err:=s.Rules("google")
	if err!=nil{t.Fatal(err)}
	if len(siteRules)==0{t.Fatal("empty Geosite rules")}
	r:=siteRules[0].DefaultOptions
	if len(r.Domain)+len(r.DomainSuffix)+len(r.DomainRegex)+len(r.DomainKeyword)==0{t.Fatal("empty Geosite matchers")}
}
