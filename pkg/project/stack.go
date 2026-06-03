package project

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/sst/sst/v3/pkg/project/common"
	"github.com/sst/sst/v3/pkg/project/provider"
)

type BuildFailedEvent struct {
	Error string
}

type StackInput struct {
	Command    string
	Target     []string
	Exclude    []string
	ServerPort int
	Dev        bool
	Verbose    bool
	Continue   bool
	SkipHash   string
	PolicyPath string
}

type ConcurrentUpdateEvent struct{}

type CancelledEvent struct{}

type BuildSuccessEvent struct {
	Files []string
	Hash  string
}

type SkipEvent struct {
}

type PolicyAdvisoryEvent struct {
	Policy  string
	Message string
	URN     string
}

type Dev struct {
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Directory   string            `json:"directory"`
	Autostart   bool              `json:"autostart"`
	Links       []string          `json:"links"`
	Title       string            `json:"title"`
	Environment map[string]string `json:"environment"`
	Aws         *struct {
		Role string `json:"role"`
	} `json:"aws"`
	Cloudflare *DevCloudflare `json:"cloudflare"`
}
type Devs map[string]Dev

type DevCloudflare struct {
	Path string `json:"path"`
}

type Task struct {
	Name      string  `json:"-"`
	Command   *string `json:"command"`
	Directory string  `json:"directory"`
}

type CompleteEvent struct {
	UpdateID    string
	Links       common.Links
	Devs        Devs
	Tasks       map[string]Task
	Outputs     map[string]interface{}
	Hints       map[string]string
	Versions    map[string]int
	Errors      []Error
	Finished    bool
	Old         bool
	Resources   []apitype.ResourceV3
	ImportDiffs map[string][]ImportDiff
	Tunnels     map[string]Tunnel
}

type Tunnel struct {
	IP         string   `json:"ip"`
	Username   string   `json:"username"`
	PrivateKey string   `json:"privateKey"`
	Subnets    []string `json:"subnets"`
}

type ImportDiff struct {
	URN   string
	Input string
	Old   interface{}
	New   interface{}
}

type StackCommandEvent struct {
	App     string
	Stage   string
	Config  string
	Command string
	Version string
}

type Error struct {
	Message string   `json:"message"`
	URN     string   `json:"urn"`
	Help    []string `json:"help"`
}

var ErrStackRunFailed = fmt.Errorf("stack run had errors")
var ErrStageNotFound = fmt.Errorf("stage not found")
var ErrPassphraseInvalid = fmt.Errorf("passphrase invalid")
var ErrProtectedStage = fmt.Errorf("cannot remove protected stage")
var ErrProtectedDevStage = fmt.Errorf("cannot run sst dev on protected stage")
var ErrPolicyViolation = fmt.Errorf("policy violations detected")
var ErrPolicyConfigError = fmt.Errorf("policy configuration error")

func (p *Project) ResolvePolicyPackPath(policyPath string) (string, error) {
	var resolvedPath string
	if filepath.IsAbs(policyPath) {
		resolvedPath = policyPath
	} else {
		resolvedPath = filepath.Join(p.PathRoot(), policyPath)
	}

	if _, err := os.Stat(resolvedPath); err != nil {
		return "", fmt.Errorf("Policy pack not found in path: %v", resolvedPath)
	}

	return resolvedPath, nil
}

func (p *Project) Lock(command string) (*provider.Update, error) {
	return provider.Lock(p.home, p.Version(), command, p.app.Name, p.app.Stage)
}

func (s *Project) Unlock() error {
	return provider.Unlock(s.home, s.version, s.app.Name, s.app.Stage)
}

func (s *Project) ForceUnlock() error {
	return provider.ForceUnlock(s.home, s.version, s.app.Name, s.app.Stage)
}

func getNotNilFields(v interface{}) []interface{} {
	result := []interface{}{}
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		result = append(result, v)
		return result
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		switch field.Kind() {
		case reflect.Struct:
			result = append(result, getNotNilFields(field.Interface())...)
			break
		case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
			if !field.IsNil() {
				result = append(result, field.Interface())
			}
			break
		}
	}

	return result
}
