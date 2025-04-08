package project

import (
	"fmt"
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
	ServerPort int
	Dev        bool
	Verbose    bool
	Continue   bool
	SkipHash   string
}

type ConcurrentUpdateEvent struct{}

type CancelledEvent struct{}

type BuildSuccessEvent struct {
	Files []string
	Hash  string
}

type SkipEvent struct {
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
}
type Devs map[string]Dev

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

type CommonError struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Short   []string `json:"short"`
	Long    []string `json:"long"`
}

var CommonErrors = []CommonError{
	{
		Code:    "TooManyCacheBehaviors",
		Message: "TooManyCacheBehaviors: Your request contains more CacheBehaviors than are allowed per distribution",
		Short: []string{
			"There are too many top-level files and directories inside your app's public asset directory. Move some of them inside subdirectories.",
			"Learn more about this https://sst.dev/docs/common-errors#toomanycachebehaviors",
		},
		Long: []string{
			"This error usually happens to `SvelteKit`, `SolidStart`, `Nuxt`, and `Analog` components.",
			"",
			"CloudFront distributions have a **limit of 25 cache behaviors** per distribution. Each top-level file or directory in your frontend app's asset directory creates a cache behavior.",
			"",
			"For example, in the case of SvelteKit, the static assets are in the `static/` directory. If you have a file and a directory in it, it'll create 2 cache behaviors.",
			"",
			"```bash frame=\"none\"",
			"static/",
			"├── icons/       # Cache behavior for /icons/*",
			"└── logo.png     # Cache behavior for /logo.png",
			"```",
			"So if you have many of these at the top-level, you'll hit the limit. You can request a limit increase through the AWS Support.",
			"",
			"Alternatively, you can move some of these into subdirectories. For example, moving them to an `images/` directory, will only create 1 cache behavior.",
			"",
			"```bash frame=\"none\"",
			"static/",
			"└── images/      # Cache behavior for /images/*",
			"    ├── icons/",
			"    └── logo.png",
			"```",
			"Learn more about these [CloudFront limits](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cloudfront-limits.html#limits-web-distributions).",
		},
	},
}

var ErrStackRunFailed = fmt.Errorf("stack run had errors")
var ErrStageNotFound = fmt.Errorf("stage not found")
var ErrPassphraseInvalid = fmt.Errorf("passphrase invalid")
var ErrProtectedStage = fmt.Errorf("cannot remove protected stage")

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
