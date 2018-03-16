package app

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const (
	kernelUUIDPath     = "/proc/sys/kernel/random/uuid"
	cgroupMemLimitPath = "/sys/fs/cgroup/memory/memory.limit_in_bytes"
	cgroupMemUnlimited = 9223372036854771712
)

type App struct {
	Env func(string) (string, bool)

	name       string
	mem        uint64
	disk       uint64
	fds        uint64
	id         string
	instanceID string
	spaceID    string
	version    string
	ip         string
}

func New() (*App, error) {
	var err error
	app := &App{Env: os.LookupEnv}
	app.name = "app"
	if app.mem, err = totalMem(); err != nil {
		return nil, err
	}
	app.disk = 1024
	var fds syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &fds); err != nil {
		return nil, err
	}
	app.fds = fds.Cur
	if app.ip, err = containerIP(); err != nil {
		return nil, err
	}
	for _, id := range []*string{&app.id, &app.instanceID, &app.spaceID, &app.version} {
		if *id, err = uuid(); err != nil {
			return nil, err
		}
	}
	return app, nil
}

func containerIP() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ip := addr.To4(); ip != nil {
			return ip.String(), nil
		}
	}
	return "", errors.New("no valid ipv4 address found")
}

func uuid() (string, error) {
	id, err := ioutil.ReadFile(kernelUUIDPath)
	return strings.TrimSpace(string(id)), err
}

func totalMem() (uint64, error) {
	contents, err := ioutil.ReadFile(cgroupMemLimitPath)
	if err != nil {
		return 0, err
	}
	memBytes, err := strconv.ParseUint(strings.TrimSpace(string(contents)), 10, 64)
	if err != nil {
		return 0, err
	}
	if memBytes == cgroupMemUnlimited {
		return 1024, nil
	}
	return memBytes / 1024 / 1024, nil
}

func (a *App) config() (name, uri string, limits map[string]uint64) {
	name = a.envStr("PACK_APP_NAME", a.name)
	uri = a.envStr("PACK_APP_URI", name+".local")

	disk := a.envInt("PACK_APP_DISK", a.disk)
	fds := a.envInt("PACK_APP_FDS", a.fds)
	mem := a.envInt("PACK_APP_MEM", a.mem)
	limits = map[string]uint64{"disk": disk, "fds": fds, "mem": mem}

	return name, uri, limits
}

func (a *App) Stage() map[string]string {
	sysEnv := map[string]string{
		"HOME":   "/app",
		"LANG":   "en_US.UTF-8",
		"PATH":   "/usr/local/bin:/usr/bin:/bin",
		"TMPDIR": "/tmp",
		"USER":   "heroku",
	}

	appEnv := map[string]string{
		"STACK":                   "heroku-16",
		"DYNO":                    "local.1",
	}
	a.envOverride(appEnv)

	return mergeMaps(sysEnv, appEnv)
}

func (a *App) Launch() map[string]string {
	sysEnv := map[string]string{
		"HOME":   "/app",
		"LANG":   "en_US.UTF-8",
		"PATH":   "/usr/local/bin:/usr/bin:/bin",
		"TMPDIR": "/tmp",
		"USER":   "heroku",
	}

	appEnv := map[string]string{
		"PORT":                    "5000",
		"STACK":                   "heroku-16",
		"DYNO":                    "local.1",
	}
	a.envOverride(appEnv)

	return mergeMaps(sysEnv, appEnv)
}

func (a *App) envStr(key, val string) string {
	if v, ok := a.Env(key); ok {
		return v
	}
	return val
}

func (a *App) envInt(key string, val uint64) uint64 {
	if v, ok := a.Env(key); ok {
		if vInt, err := strconv.ParseUint(v, 10, 64); err == nil {
			return vInt
		}
	}
	return val
}

func (a *App) envOverride(m map[string]string) {
	for k, v := range m {
		m[k] = a.envStr(k, v)
	}
}

func mergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}