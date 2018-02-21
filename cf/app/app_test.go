package app_test

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/sclevine/spec"

	pkgapp "github.com/sclevine/packs/cf/app"
)

func TestApp(t *testing.T) {
	spec.Run(t, "#Stage", testStage)
}

func testStage(t *testing.T, when spec.G, it spec.S) {
	it("should return the default staging env", func() {
		app, err := pkgapp.New()
		if err != nil {
			t.Fatalf("Failed to create app: %s\n", err)
		}
		actualEnv := app.Stage()
		expectedEnv := []struct {
			k, v2 string
			cmp   func(t *testing.T, v1, v2 string)
		}{
			{"CF_STACK", "cflinuxfs2", nil},
			{"USER", "vcap", nil},
			{"HOME", "/home/vcap", nil},
			{"CF_INSTANCE_IP", "container", ipCmp},
			{"VCAP_APPLICATION", "default", vcapAppCmp},
			{"CF_INSTANCE_ADDR", "", nil},
			{"CF_INSTANCE_PORT", "", nil},
			{"CF_INSTANCE_PORTS", "[]", nil},
			{"VCAP_SERVICES", "{}", nil},
			{"PATH", "/usr/local/bin:/usr/bin:/bin", nil},
			{"LANG", "en_US.UTF-8", nil},
			{"MEMORY_LIMIT", "1024m", nil},
		}
		if v1, v2 := len(actualEnv), len(expectedEnv); v1 != v2 {
			t.Fatalf("Different lengths: %d != %d\n", v1, v2)
		}
		for _, exp := range expectedEnv {
			if v1, ok := actualEnv[exp.k]; !ok {
				t.Fatalf("Missing: %s\n", exp.k)
			} else if exp.cmp != nil {
				exp.cmp(t, v1, exp.v2)
			} else if v1 != exp.v2 {
				t.Fatalf("%s: %s != %s\n", exp.k, v1, exp.v2)
			}
		}
	})
}

func ipCmp(t *testing.T, ip, kind string) {
	t.Helper()
	out, err := exec.Command("hostname", "-i").Output()
	if err != nil {
		t.Fatalf("Error: %s\n", err)
	}
	if expected := strings.TrimSpace(string(out)); ip != expected {
		t.Fatalf("Mismatched %s IP: %s != %s\n", kind, ip, expected)
	}
}

func vcapAppCmp(t *testing.T, vcapAppJSON, kind string) {
	t.Helper()
	var vcapApp, expected pkgapp.VCAPApplication
	if err := json.Unmarshal([]byte(vcapAppJSON), &vcapApp); err != nil {
		t.Fatalf("Error: %s\n", err)
	}
	for _, uuid := range []string{
		vcapApp.ApplicationID,
		vcapApp.ApplicationVersion,
		vcapApp.SpaceID,
		vcapApp.Version,
	} {
		if len(uuid) != 36 {
			t.Fatalf("Invalid UUID: %s\n", uuid)
		}
	}
	if v, av := vcapApp.Version, vcapApp.ApplicationVersion; v != av {
		t.Fatalf("Mismatched version UUIDs: %s != %s\n", v, av)
	}
	ulimit, err := exec.Command("bash", "-c", "ulimit -n").Output()
	if err != nil {
		t.Fatalf("Error: %s\n", err)
	}
	fds, err := strconv.ParseUint(strings.TrimSpace(string(ulimit)), 10, 64)
	if err != nil {
		t.Fatalf("Error: %s\n", err)
	}
	if kind == "default" {
		expected = pkgapp.VCAPApplication{
			ApplicationID:      vcapApp.ApplicationID,
			ApplicationName:    "app",
			ApplicationURIs:    []string{"app.local"},
			ApplicationVersion: vcapApp.ApplicationVersion,
			Limits:             map[string]uint64{"disk": 1024, "fds": fds, "mem": 1024},
			Name:               "app",
			SpaceID:            vcapApp.SpaceID,
			SpaceName:          "app-space",
			URIs:               []string{"app.local"},
			Version:            vcapApp.Version,
		}
	} else {
		t.Fatalf("Invalid kind: %s\n", kind)
	}

	if !reflect.DeepEqual(vcapApp, expected) {
		t.Fatalf("Mismatched %s:\n%#v\n!=\n%#v\n", kind, vcapApp, expected)
	}
}
