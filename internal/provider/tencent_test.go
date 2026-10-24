package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
)

const (
	tcPlanChat = "https://api.lkeap.cloud.tencent.com/plan/v3"
	tcPlanMsgs = "https://api.lkeap.cloud.tencent.com/plan/anthropic"
	tcPlanKeys = "https://console.cloud.tencent.com/tokenhub/tokenplan"
	tcPlanDocs = "https://cloud.tencent.com/document/product/1823/130060"
	tcCNHost   = "https://tokenhub.tencentmaas.com"
	tcCNKeys   = "https://console.cloud.tencent.com/tokenhub/apikey"
	tcCNDocs   = "https://cloud.tencent.com/document/product/1823/130078"
	tcIntlHost = "https://tokenhub-intl.tencentmaas.com"
	tcIntlKeys = "https://console.tencentcloud.com/tokenhub/apikey"
	tcIntlDocs = "https://www.tencentcloud.com/document/product/1300/78939"
)

// tcRegions are Tencent Cloud's three, as the three presets they were
// before (Jorben on Discord) gave them: the old id, the region it is now,
// and what a provider there is made with.
var tcRegions = []struct {
	old, region                string
	chat, responses, anthropic string
	catalog, keys, website     string
}{
	{"tencent-token-plan", "plan", tcPlanChat, "", tcPlanMsgs, "", tcPlanKeys, tcPlanDocs},
	{"tencent-tokenhub-cn", "cn", tcCNHost + "/v1", tcCNHost + "/v1", tcCNHost, "tencent-tokenhub", tcCNKeys, tcCNDocs},
	{"tencent-tokenhub", "intl", tcIntlHost + "/v1", tcIntlHost + "/v1", tcIntlHost, "tencent-tokenhub", tcIntlKeys, tcIntlDocs},
}

// Token Plan 企业版 (#1574) is a region of Tencent Cloud's in China and
// one in Singapore, at TokenHub's hosts under /plan with chat completions,
// Responses and Anthropic messages, its own models before a list: an
// entry imported at its endpoints, a link and a provider moved there in
// the editor each land on it, and TokenHub's pay as you go at the same
// hosts stays where it was.
func TestTencentCloudEnterprisePlan(t *testing.T) {
	pr := Preset("tencent-cloud")
	for _, c := range []struct{ region, host, keys string }{
		{"enterprise", tcCNHost, tcPlanKeys},
		{"enterprise-intl", tcIntlHost, "https://console.tencentcloud.com/tokenhub/tokenplan"},
	} {
		var r *Region
		for i := range pr.Regions {
			if pr.Regions[i].ID == c.region {
				r = &pr.Regions[i]
			}
		}
		if r == nil || r.Chat != c.host+"/plan/v3" || r.Responses != c.host+"/plan/v3" || r.Anthropic != c.host+"/plan/anthropic" ||
			r.Catalog != "" || r.KeysURL != c.keys || r.Website != "https://cloud.tencent.com/document/product/1823/130659" ||
			len(r.Models) == 0 || r.Models[0] != "auto" {
			t.Fatalf("%s: %+v", c.region, r)
		}
		for _, id := range r.Models {
			if id != strings.ToLower(id) {
				t.Fatalf("%s: model ids are lowercase: %q", c.region, id)
			}
		}
		for _, e := range []endpoints{{chat: r.Chat}, {responses: r.Responses}, {anthropic: r.Anthropic}} {
			im, _ := imported("Tencent", "sk-x", e, nil)
			if im.Preset != "tencent-cloud" || im.KeysURL != c.keys || im.Catalog != "" {
				t.Fatalf("imported from %+v: %+v", e, im)
			}
			if got := im.regionOf(pr); got == nil || got.ID != c.region {
				t.Fatalf("imported from %+v at %+v", e, got)
			}
		}
		p, err := ParseImport("magpie://import?preset=tencent-cloud&region=" + c.region)
		if err != nil || p.Chat != r.Chat || p.Responses != r.Responses || p.Anthropic != r.Anthropic || p.KeysURL != c.keys {
			t.Fatalf("link: %v %+v", err, p)
		}
		if got := p.planModels(nil); len(got) != len(r.Models) || got[0].ID != "auto" {
			t.Fatalf("%s's models: %+v", c.region, got)
		}
		// pay as you go at the same host moved here takes the plan's key
		// page and docs, and no catalog
		q, _ := FromPreset("tencent-cloud")
		q.Chat, q.Responses, q.Anthropic, q.Catalog, q.KeysURL, q.Website = c.host+"/v1", c.host+"/v1", c.host, "tencent-tokenhub", "", ""
		if got := q.regionOf(pr); got == nil || got.ID == c.region {
			t.Fatalf("pay as you go at %+v", got)
		}
		q.Chat, q.Responses, q.Anthropic, q.Catalog = r.Chat, r.Responses, r.Anthropic, ""
		if n := normalize(q); n.KeysURL != c.keys || n.Website != r.Website || n.Catalog != "" {
			t.Fatalf("moved to %s: %+v", c.region, n)
		}
	}
}

// A provider added under any of the three presets Tencent Cloud was keeps
// working with nothing done: its id, key, endpoints, models, catalog
// (names and prices), key page and docs as they were, the routing group
// naming it finding it, and the add sheet counting it as Tencent Cloud's;
// a save from the editor (which sends no key page or docs) keeps them,
// and moving it to another region takes that region's.
func TestTencentCloudOldPresetsKeepWorking(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	// providers.json as an older magpie wrote it
	old := `{"providers":[
{"id":"tencent-token-plan","name":"Tencent Cloud Token Plan","icon":"tencentcloud-color","preset":"tencent-token-plan","key":"sk-tp-plan",
 "chat":"` + tcPlanChat + `","anthropic":"` + tcPlanMsgs + `","models":["glm-5.3","hy3"],"website":"` + tcPlanDocs + `","keysUrl":"` + tcPlanKeys + `"},
{"id":"tencent-tokenhub-cn","name":"Tencent Cloud TokenHub (China)","icon":"tencentcloud-color","preset":"tencent-tokenhub-cn","key":"sk-cn",
 "chat":"` + tcCNHost + `/v1","responses":"` + tcCNHost + `/v1","anthropic":"` + tcCNHost + `","models":["hy3","deepseek-v4-pro"],"catalog":"tencent-tokenhub","website":"` + tcCNDocs + `","keysUrl":"` + tcCNKeys + `"},
{"id":"tencent-tokenhub","name":"Tencent Cloud TokenHub","icon":"tencentcloud-color","preset":"tencent-tokenhub","key":"sk-intl",
 "chat":"` + tcIntlHost + `/v1","responses":"` + tcIntlHost + `/v1","anthropic":"` + tcIntlHost + `","models":["hy3"],"catalog":"tencent-tokenhub","website":"` + tcIntlDocs + `","keysUrl":"` + tcIntlKeys + `"}],
"groups":[{"id":"hy","name":"Hy","members":["tencent-token-plan/hy3","tencent-tokenhub-cn/hy3","tencent-tokenhub/hy3"]}]}`
	if err := os.MkdirAll(filepath.Dir(Path()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	keys := map[string]string{"tencent-token-plan": "sk-tp-plan", "tencent-tokenhub-cn": "sk-cn", "tencent-tokenhub": "sk-intl"}
	models := map[string][]string{"tencent-token-plan": {"glm-5.3", "hy3"}, "tencent-tokenhub-cn": {"hy3", "deepseek-v4-pro"}, "tencent-tokenhub": {"hy3"}}
	check := func(when string) {
		t.Helper()
		for _, c := range tcRegions {
			p, err := Find(c.old)
			if err != nil {
				t.Fatalf("%s: %s: %v", when, c.old, err)
			}
			if p.ID != c.old || p.Preset != "tencent-cloud" || p.Key != keys[c.old] || !slices.Equal(p.Models, models[c.old]) ||
				p.Chat != c.chat || p.Responses != c.responses || p.Anthropic != c.anthropic ||
				p.Catalog != c.catalog || p.KeysURL != c.keys || p.Website != c.website || p.Icon != "tencentcloud-color" {
				t.Fatalf("%s: %s: %+v", when, c.old, p)
			}
			// the plan's own models before its list, TokenHub none of them
			if got := p.planModels(nil); (c.region == "plan") != (len(got) > 0) {
				t.Fatalf("%s: %s's models: %+v", when, c.old, got)
			}
			// its region, as the editor shows it
			if r := p.regionOf(Preset(p.Preset)); r == nil || r.ID != c.region {
				t.Fatalf("%s: %s at %+v", when, c.old, r)
			}
		}
		// the routing group naming each by its id finds all three
		_, ms, ok := FindGroup(GroupPrefix + "hy")
		if !ok || len(ms) != 3 {
			t.Fatalf("%s: group: %v %+v", when, ok, ms)
		}
		for i, c := range tcRegions {
			if ms[i].Provider.ID != c.old || ms[i].Model != "hy3" || ms[i].Provider.Chat != c.chat {
				t.Fatalf("%s: member %d: %+v", when, i, ms[i])
			}
		}
	}
	check("read")
	// the editor's Save: the provider's own fields, no key page or docs
	for _, c := range tcRegions {
		p, _ := Find(c.old)
		p.Website, p.KeysURL = "", ""
		if err := Save(*p); err != nil {
			t.Fatal(err)
		}
	}
	check("saved")

	// a plan moved to pay as you go in the editor takes TokenHub's key
	// page, docs and catalog, and back again the plan's
	p, _ := Find("tencent-token-plan")
	p.Chat, p.Responses, p.Anthropic, p.Website, p.KeysURL = tcIntlHost+"/v1", tcIntlHost+"/v1", tcIntlHost, "", ""
	if n := normalize(*p); n.Catalog != "tencent-tokenhub" || n.KeysURL != tcIntlKeys || n.Website != tcIntlDocs {
		t.Fatalf("to pay as you go: %+v", n)
	}
	p, _ = Find("tencent-tokenhub-cn")
	p.Chat, p.Responses, p.Anthropic, p.Website, p.KeysURL = tcPlanChat, "", tcPlanMsgs, "", ""
	if n := normalize(*p); n.Catalog != "" || n.KeysURL != tcPlanKeys || n.Website != tcPlanDocs {
		t.Fatalf("to the plan: %+v", n)
	}
	// a catalog the user gave stays wherever it is
	p.Catalog = "deepseek"
	if n := normalize(*p); n.Catalog != "deepseek" {
		t.Fatalf("own catalog: %q", n.Catalog)
	}
}

// TokenHub's /v1/models, as its docs show it, asked with the key as a
// Bearer: its language models are kept, the embedding one left out.
func TestTencentTokenHubList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer sk-test" {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.Write([]byte(`{"object":"list","data":[
{"id":"hy3","object":"model","name":"Hy3","created":1783267200,"status":"online"},
{"id":"deepseek-v4-pro","object":"model","name":"DeepSeek-V4-Pro","created":1776960000,"status":"online"},
{"id":"kinfra-text-embedding-0.6b","object":"model","name":"KInfra Embedding","created":1776960000,"status":"online"}]}`))
	}))
	defer srv.Close()
	ms, _, err := catalog.FetchAt(context.Background(), srv.URL+"/v1", "sk-test", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].ID != "hy3" || ms[1].ID != "deepseek-v4-pro" {
		t.Fatalf("%+v", ms)
	}
}
