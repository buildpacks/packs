package app_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/sclevine/spec"

	pkgapp "github.com/sclevine/packs/cf/app"
)

var memory = flag.Uint64("memory", 1024, "expected memory usage in mb")

type cmpFunc func(t *testing.T, v1, v2 string)

func TestApp(t *testing.T) {
	spec.Run(t, "#Stage", testStage)
	spec.Run(t, "#Launch", testLaunch)
}

func testStage(t *testing.T, when spec.G, it spec.S) {
	var (
		app *pkgapp.App
		set func(k, v string)
		reset func()
	)

	it.Before(func() {
		var err error
		if app, err = pkgapp.New(); err != nil {
			t.Fatalf("Failed to create app: %s\n", err)
		}
		set, reset = setEnv(t)
	})

	it.After(func() {
		reset()
	})

	it("should return the default staging env", func() {
		env := app.Stage()
		vcapApp, err := vcapAppExpect(env["VCAP_APPLICATION"])
		if err != nil {
			t.Fatalf("Error: %s\n", err)
		}
		vcapAppJSON, err := json.Marshal(vcapApp)
		if err != nil {
			t.Fatalf("Error: %s\n", err)
		}
		expectedEnv := []struct {
			k, v2 string
			cmp   cmpFunc
		}{
			{"CF_INSTANCE_ADDR", "", nil},
			{"CF_INSTANCE_IP", "", ipCmp},
			{"CF_INSTANCE_PORT", "", nil},
			{"CF_INSTANCE_PORTS", "[]", nil},
			{"CF_STACK", "cflinuxfs2", nil},
			{"HOME", "/home/vcap", nil},
			{"LANG", "en_US.UTF-8", nil},
			{"MEMORY_LIMIT", fmt.Sprintf("%dm", *memory), nil},
			{"PATH", "/usr/local/bin:/usr/bin:/bin", nil},
			{"USER", "vcap", nil},
			{"VCAP_APPLICATION", string(vcapAppJSON), vcapAppCmp},
			{"VCAP_SERVICES", "{}", nil},
		}
		if v1, v2 := len(env), len(expectedEnv); v1 != v2 {
			t.Fatalf("Different lengths: %d != %d\n", v1, v2)
		}
		for _, exp := range expectedEnv {
			if v1, ok := env[exp.k]; !ok {
				t.Fatalf("Missing: %s\n", exp.k)
			} else if exp.cmp != nil {
				exp.cmp(t, v1, exp.v2)
			} else if v1 != exp.v2 {
				t.Fatalf("%s: %s != %s\n", exp.k, v1, exp.v2)
			}
		}
	})

	when("all custom env variables are set", func() {
		it("should return a custom staging env", func() {
			set("PACK_APP_NAME", "some-name")
			set("PACK_APP_URI", "some-uri")
			set("PACK_APP_DISK", "10")
			set("PACK_APP_FDS", "20")
			set("PACK_APP_MEM", "30")

			env := app.Stage()
			if mem := env["MEMORY_LIMIT"]; mem != "30m" {
				t.Fatalf("Incorrect memory: %d", mem)
			}

			vcapApp, err := vcapAppExpect(env["VCAP_APPLICATION"])
			if err != nil {
				t.Fatalf("Error: %s\n", err)
			}
			vcapApp.ApplicationName = "some-name"
			vcapApp.ApplicationURIs = []string{"some-uri"}
			vcapApp.Limits = map[string]uint64{"disk": 10, "fds": 20, "mem": 30}
			vcapApp.Name = "some-name"
			vcapApp.SpaceName = "some-name-space"
			vcapApp.URIs = []string{"some-uri"}
			vcapAppJSON, err := json.Marshal(vcapApp)
			if err != nil {
				t.Fatalf("Error: %s\n", err)
			}
			vcapAppCmp(t, env["VCAP_APPLICATION"], string(vcapAppJSON))
		})
	})

	when("only a custom name is set", func() {
		it("should use the name for the uri as well", func() {
			set("PACK_APP_NAME", "some-name")

			env := app.Stage()

			vcapApp, err := vcapAppExpect(env["VCAP_APPLICATION"])
			if err != nil {
				t.Fatalf("Error: %s\n", err)
			}
			vcapApp.ApplicationName = "some-name"
			vcapApp.ApplicationURIs = []string{"some-name.local"}
			vcapApp.Name = "some-name"
			vcapApp.SpaceName = "some-name-space"
			vcapApp.URIs = []string{"some-name.local"}
			vcapAppJSON, err := json.Marshal(vcapApp)
			if err != nil {
				t.Fatalf("Error: %s\n", err)
			}
			vcapAppCmp(t, env["VCAP_APPLICATION"], string(vcapAppJSON))
		})
	})
}

func testLaunch(t *testing.T, when spec.G, it spec.S) {
	var (
		app *pkgapp.App
		set func(k, v string)
		reset func()
	)

	it.Before(func() {
		var err error
		if app, err = pkgapp.New(); err != nil {
			t.Fatalf("Failed to create app: %s\n", err)
		}
		set, reset = setEnv(t)
	})

	it.After(func() {
		reset()
	})

	it("should return the default launch env", func() {
		env := app.Launch()
		vcapApp, err := vcapAppExpect(env["VCAP_APPLICATION"])
		if err != nil {
			t.Fatalf("Error: %s\n", err)
		}
		vcapApp.Host = "0.0.0.0"
		vcapApp.InstanceID = env["CF_INSTANCE_GUID"]
		vcapApp.InstanceIndex = uintPtr(0)
		vcapApp.Port = uintPtr(8080)
		vcapAppJSON, err := json.Marshal(vcapApp)
		if err != nil {
			t.Fatalf("Error: %s\n", err)
		}
		expectedEnv := []struct {
			k, v2 string
			cmp   cmpFunc
		}{
			{"CF_INSTANCE_ADDR", ":8080", ipCmp},
			{"CF_INSTANCE_GUID", "", uuidCmp},
			{"CF_INSTANCE_INDEX", "0", nil},
			{"CF_INSTANCE_IP", "", ipCmp},
			{"CF_INSTANCE_PORT", "8080", nil},
			{"CF_INSTANCE_PORTS", `[{"external":8080,"internal":8080}]`, nil},
			{"HOME", "/home/vcap/app", nil},
			{"INSTANCE_INDEX", "0", nil},
			{"LANG", "en_US.UTF-8", nil},
			{"MEMORY_LIMIT", fmt.Sprintf("%dm", *memory), nil},
			{"PATH", "/usr/local/bin:/usr/bin:/bin", nil},
			{"PORT", "8080", nil},
			{"TMPDIR", "/home/vcap/tmp", nil},
			{"USER", "vcap", nil},
			{"VCAP_APPLICATION", string(vcapAppJSON), vcapAppCmp},
			{"VCAP_SERVICES", "{}", nil},
		}
		if v1, v2 := len(env), len(expectedEnv); v1 != v2 {
			t.Fatalf("Different lengths: %d != %d\n", v1, v2)
		}
		for _, exp := range expectedEnv {
			if v1, ok := env[exp.k]; !ok {
				t.Fatalf("Missing: %s\n", exp.k)
			} else if exp.cmp != nil {
				exp.cmp(t, v1, exp.v2)
			} else if v1 != exp.v2 {
				t.Fatalf("%s: %s != %s\n", exp.k, v1, exp.v2)
			}
		}
	})

	when("all custom env variables are set", func() {
		it("should return a custom staging env", func() {
			set("PACK_APP_NAME", "some-name")
			set("PACK_APP_URI", "some-uri")
			set("PACK_APP_DISK", "10")
			set("PACK_APP_FDS", "20")
			set("PACK_APP_MEM", "30")

			env := app.Launch()
			if mem := env["MEMORY_LIMIT"]; mem != "30m" {
				t.Fatalf("Incorrect memory: %d", mem)
			}

			vcapApp, err := vcapAppExpect(env["VCAP_APPLICATION"])
			if err != nil {
				t.Fatalf("Error: %s\n", err)
			}

			vcapApp.Host = "0.0.0.0"
			vcapApp.InstanceID = env["CF_INSTANCE_GUID"]
			vcapApp.InstanceIndex = uintPtr(0)
			vcapApp.Port = uintPtr(8080)

			vcapApp.ApplicationName = "some-name"
			vcapApp.ApplicationURIs = []string{"some-uri"}
			vcapApp.Limits = map[string]uint64{"disk": 10, "fds": 20, "mem": 30}
			vcapApp.Name = "some-name"
			vcapApp.SpaceName = "some-name-space"
			vcapApp.URIs = []string{"some-uri"}

			vcapAppJSON, err := json.Marshal(vcapApp)
			if err != nil {
				t.Fatalf("Error: %s\n", err)
			}
			vcapAppCmp(t, env["VCAP_APPLICATION"], string(vcapAppJSON))
		})
	})

	when("only a custom name is set", func() {
		it("should use the name for the uri as well", func() {
			set("PACK_APP_NAME", "some-name")

			env := app.Launch()

			vcapApp, err := vcapAppExpect(env["VCAP_APPLICATION"])
			if err != nil {
				t.Fatalf("Error: %s\n", err)
			}

			vcapApp.Host = "0.0.0.0"
			vcapApp.InstanceID = env["CF_INSTANCE_GUID"]
			vcapApp.InstanceIndex = uintPtr(0)
			vcapApp.Port = uintPtr(8080)

			vcapApp.ApplicationName = "some-name"
			vcapApp.ApplicationURIs = []string{"some-name.local"}
			vcapApp.Name = "some-name"
			vcapApp.SpaceName = "some-name-space"
			vcapApp.URIs = []string{"some-name.local"}

			vcapAppJSON, err := json.Marshal(vcapApp)
			if err != nil {
				t.Fatalf("Error: %s\n", err)
			}
			vcapAppCmp(t, env["VCAP_APPLICATION"], string(vcapAppJSON))
		})
	})
}

func uuidCmp(t *testing.T, uuid, _ string) {
	t.Helper()
	if len(uuid) != 36 {
		t.Fatalf("Invalid UUID: %s\n", uuid)
	}
}

func ipCmp(t *testing.T, ip, suffix string) {
	t.Helper()
	out, err := exec.Command("hostname", "-i").Output()
	if err != nil {
		t.Fatalf("Error: %s\n", err)
	}
	if expected := strings.TrimSpace(string(out)) + suffix; ip != expected {
		t.Fatalf("Mismatched IP: %s != %s\n", ip, expected)
	}
}

func vcapAppExpect(vcapAppJSON string) (pkgapp.VCAPApplication, error) {
	var vcapApp pkgapp.VCAPApplication
	if err := json.Unmarshal([]byte(vcapAppJSON), &vcapApp); err != nil {
		return pkgapp.VCAPApplication{}, err
	}
	ulimit, err := exec.Command("bash", "-c", "ulimit -n").Output()
	if err != nil {
		return pkgapp.VCAPApplication{}, err
	}
	fds, err := strconv.ParseUint(strings.TrimSpace(string(ulimit)), 10, 64)
	if err != nil {
		return pkgapp.VCAPApplication{}, err
	}
	return pkgapp.VCAPApplication{
		ApplicationID:      vcapApp.ApplicationID,
		ApplicationName:    "app",
		ApplicationURIs:    []string{"app.local"},
		ApplicationVersion: vcapApp.ApplicationVersion,
		Limits:             map[string]uint64{"disk": 1024, "fds": fds, "mem": *memory},
		Name:               "app",
		SpaceID:            vcapApp.SpaceID,
		SpaceName:          "app-space",
		URIs:               []string{"app.local"},
		Version:            vcapApp.Version,
	}, nil
}

func vcapAppCmp(t *testing.T, va1, va2 string) {
	t.Helper()
	var vcapApp1 pkgapp.VCAPApplication
	if err := json.Unmarshal([]byte(va1), &vcapApp1); err != nil {
		t.Fatalf("Error: %s\n", err)
	}
	var vcapApp2 pkgapp.VCAPApplication
	if err := json.Unmarshal([]byte(va2), &vcapApp2); err != nil {
		t.Fatalf("Error: %s\n", err)
	}
	if !reflect.DeepEqual(vcapApp1, vcapApp2) {
		t.Fatalf("Mismatched VCAP_APPLICATION:\n%#v\n!=\n%#v\n", vcapApp1, vcapApp2)
	}

	set := map[string]struct{}{}
	total := 0
	for _, uuid := range []string{
		vcapApp1.ApplicationID,
		vcapApp1.SpaceID,
		vcapApp1.Version,
		vcapApp1.InstanceID,
	} {
		if uuid != "" {
			uuidCmp(t, uuid, "")
			set[uuid] = struct{}{}
			total++
		}
	}
	if l := len(set); l != total {
		t.Fatalf("Duplicate UUIDs: %d\n", total-l)
	}
	if v1, v2 := vcapApp1.Version, vcapApp2.ApplicationVersion; v1 != v2 {
		t.Fatalf("Mismatched version UUIDs: %s != %s\n", v1, v2)
	}
}

func uintPtr(i uint) *uint {
	return &i
}

func setEnv(t *testing.T) (set func(k, v string), reset func()) {
	var new []string
	saved := map[string]string{}
	return func(k, v string) {
			if old, ok := os.LookupEnv(k); ok {
				saved[k] = old
			} else {
				new = append(new, k)
			}
			if err := os.Setenv(k, v); err != nil {
				t.Fatalf("Failed to set %s=%s", k, v)
			}
		}, func() {
			for k, v := range saved {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to reset %s=%s", k, v)
				}
				delete(saved, k)
			}
			for _, k := range new {
				if err := os.Unsetenv(k); err != nil {
					t.Fatalf("Failed to unset %s", k)
				}
			}
			new = nil
		}
}
