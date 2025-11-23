package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"

	"cloud.google.com/go/compute/metadata"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	"github.com/sst/sst/v3/internal/util"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCPProvider struct {
	project     string
	region      string
	zone        string
	credentials *google.Credentials
}

func (g *GCPProvider) Env() (map[string]string, error) {
	return map[string]string{
		"GOOGLE_PROJECT": g.project,
		"GOOGLE_REGION":  g.region,
		"GOOGLE_ZONE":    g.zone,
		// fixme: pulumi and go sdk share the same underlying auth mechanism
		// so we can just assume the pulumi uses works without any special auth vars
		// but should confirm with dax
	}, nil
}

func (g *GCPProvider) Init(app, stage string, args map[string]any) error {
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return util.NewReadableError(err, "Failed to find GCP credentials, pleasu use gcloud cli to login with application default credentials.")
	}

	project := firstNonEmptyEnv(
		"GOOGLE_PROJECT",
		"GOOGLE_CLOUD_PROJECT",
		"GCLOUD_PROJECT",
		"CLOUDSDK_CORE_PROJECT",
	)
	if args["project"] != nil {
		project = args["project"].(string)
	}
	region := firstNonEmptyEnv(
		"GOOGLE_REGION",
		"GCLOUD_REGION",
		"CLOUDSDK_COMPUTE_REGION",
		"GOOGLE_CLOUD_REGION",
	)
	if r, ok := args["region"].(string); ok && r != "" {
		region = r
	}
	zone := firstNonEmptyEnv(
		"GOOGLE_ZONE",
		"GCLOUD_ZONE",
		"CLOUDSDK_COMPUTE_ZONE",
		"GOOGLE_CLOUD_ZONE",
	)
	if z, ok := args["zone"].(string); ok && z != "" {
		zone = z
	}

	if metadata.OnGCE() {
		client := metadata.NewClient(nil)
		if project == "" {
			if pid, err := client.ProjectIDWithContext(ctx); err == nil {
				project = pid
			}
		}

		if region == "" || zone == "" {
			if z, err := client.ZoneWithContext(ctx); err == nil {
				if zone == "" {
					zone = z
				}

				// z :: us-central1-a
				if region == "" {
					if dash := strings.LastIndex(z, "-"); dash != -1 {
						region = z[:dash]
					}
				}
			}
		}
	}

	if project == "" {
		return util.NewReadableError(nil, "GCP project not found. Please use GOOGLE_PROJECT environment variable or in the provider section of the project configuration file.")
	}

	// fixme: if region is still empty, should we set it to a default value?
	// some resources don't require region to be set, and pulumi/go sdk defaults to us
	// should ask dax

	g.project = project
	g.region = region
	g.credentials = creds
	slog.Info("gcp project selected", "project", project)
	return nil
}

type GCPHome struct {
	provider        *GCPProvider
	bootstrapBucket string
	gcsClient       *storage.Client
	secretClient    *secretmanager.Client
	once            sync.Once
	initErr         error
}

func NewGCPHome(provider *GCPProvider) *GCPHome {
	return &GCPHome{
		provider: provider,
	}
}

func (g *GCPHome) initClients() {
	g.once.Do(func() {
		ctx := context.Background()

		gcsClient, err := storage.NewClient(ctx, option.WithCredentials(g.provider.credentials))
		if err != nil {
			g.initErr = fmt.Errorf("failed to create GCS client: %w", err)
			return
		}

		secretClient, err := secretmanager.NewClient(ctx, option.WithCredentials(g.provider.credentials))
		if err != nil {
			gcsClient.Close()
			g.initErr = fmt.Errorf("failed to create Secret Manager client: %w", err)
			return
		}

		g.gcsClient = gcsClient
		g.secretClient = secretClient
	})
}

func (g *GCPHome) getGCSClient() (*storage.Client, error) {
	g.initClients()
	if g.initErr != nil {
		return nil, g.initErr
	}
	if g.gcsClient == nil {
		return nil, fmt.Errorf("GCS client not initialized")
	}
	return g.gcsClient, nil
}

func (g *GCPHome) getSecretClient() (*secretmanager.Client, error) {
	g.initClients()
	if g.initErr != nil {
		return nil, g.initErr
	}
	if g.secretClient == nil {
		return nil, fmt.Errorf("Secret Manager client not initialized")
	}
	return g.secretClient, nil
}

func (g *GCPHome) Bootstrap() error {
	gcsClient, err := g.getGCSClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// fixme: projectId is unique, so sst-state-{projectId} should be unique
	// should be fune but should ask dax anyway
	bucketName := fmt.Sprintf("sst-state-%s", g.provider.project)
	bucket := gcsClient.Bucket(bucketName)

	_, err = bucket.Attrs(ctx)
	if errors.Is(err, storage.ErrBucketNotExist) {
		slog.Info("creating new bucket", "bucket", bucketName)
		if err := bucket.Create(
			ctx,
			g.provider.project,
			&storage.BucketAttrs{
				Location: g.provider.region,
				// just in case pulumi corrupts the state or something idk
				VersioningEnabled: true,
			},
		); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	slog.Info("found existing bucket", "bucket", bucketName)
	g.bootstrapBucket = bucketName

	return nil
}

func (g *GCPHome) getData(key, app, stage string) (io.Reader, error) {
	gcsClient, err := g.getGCSClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	bucket := gcsClient.Bucket(g.bootstrapBucket)

	obj := bucket.Object(g.pathForData(key, app, stage))
	r, err := obj.NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

func (g *GCPHome) putData(key, app, stage string, data io.Reader) error {
	gcsClient, err := g.getGCSClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	bucket := gcsClient.Bucket(g.bootstrapBucket)

	obj := bucket.Object(g.pathForData(key, app, stage))
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, data); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

func (g *GCPHome) removeData(key, app, stage string) error {
	gcsClient, err := g.getGCSClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	bucket := gcsClient.Bucket(g.bootstrapBucket)

	obj := bucket.Object(g.pathForData(key, app, stage))
	return obj.Delete(ctx)
}

func (g *GCPHome) cleanup(key, app, stage string) error {
	gcsClient, err := g.getGCSClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	bucket := gcsClient.Bucket(g.bootstrapBucket)

	prefix := path.Join(key, app, stage) + "/"
	slog.Info("cleaning up folder", "bucket", g.bootstrapBucket, "prefix", prefix)

	it := bucket.Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		obj := bucket.Object(attrs.Name)
		if err := obj.Delete(ctx); err != nil {
			return err
		}
	}

	slog.Info("folder cleanup complete", "prefix", prefix)
	return nil
}

func (g *GCPHome) getPassphrase(app string, stage string) (string, error) {
	secretClient, err := g.getSecretClient()
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	secretName := g.pathForPassphrase(app, stage)
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", g.provider.project, secretName)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}

	result, err := secretClient.AccessSecretVersion(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return "", nil
		}
		return "", err
	}

	return string(result.Payload.Data), nil
}

func (g *GCPHome) setPassphrase(app, stage, passphrase string) error {
	secretClient, err := g.getSecretClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	secretName := g.pathForPassphrase(app, stage)
	parent := fmt.Sprintf("projects/%s", g.provider.project)

	// Check if secret already exists
	getReq := &secretmanagerpb.GetSecretRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s", g.provider.project, secretName),
	}

	_, err = secretClient.GetSecret(ctx, getReq)
	if err != nil && !strings.Contains(err.Error(), "NotFound") {
		return err
	}

	if err != nil && strings.Contains(err.Error(), "NotFound") {
		// Create the secret
		createReq := &secretmanagerpb.CreateSecretRequest{
			Parent:   parent,
			SecretId: secretName,
			Secret: &secretmanagerpb.Secret{
				Replication: &secretmanagerpb.Replication{
					Replication: &secretmanagerpb.Replication_Automatic_{
						Automatic: &secretmanagerpb.Replication_Automatic{},
					},
				},
			},
		}

		secret, err := secretClient.CreateSecret(ctx, createReq)
		if err != nil {
			return err
		}

		// Add the secret version
		addReq := &secretmanagerpb.AddSecretVersionRequest{
			Parent: secret.Name,
			Payload: &secretmanagerpb.SecretPayload{
				Data: []byte(passphrase),
			},
		}

		_, err = secretClient.AddSecretVersion(ctx, addReq)
		return err
	}

	// Secret exists, add a new version
	addReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: fmt.Sprintf("projects/%s/secrets/%s", g.provider.project, secretName),
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(passphrase),
		},
	}

	_, err = secretClient.AddSecretVersion(ctx, addReq)
	return err
}

func (g *GCPHome) listStages(app string) ([]string, error) {
	gcsClient, err := g.getGCSClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	bucketName := fmt.Sprintf("sst-state-%s", g.provider.project)
	bucket := gcsClient.Bucket(bucketName)

	prefix := path.Join("app", app) + "/"
	it := bucket.Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	stages := []string{}
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		filename := path.Base(attrs.Name)
		if strings.HasSuffix(filename, ".json") {
			stageName := strings.TrimSuffix(filename, ".json")
			stages = append(stages, stageName)
		}
	}

	return stages, nil
}

func (g *GCPHome) info() (util.KeyValuePairs[string], error) {
	lines := util.KeyValuePairs[string]{
		{Key: "Provider", Value: "GCP"},
		{Key: "Project", Value: g.provider.project},
	}
	if g.provider.region != "" {
		lines = append(lines, util.KeyValuePair[string]{
			Key: "Region", Value: g.provider.region,
		})
	}
	if g.provider.zone != "" {
		lines = append(lines, util.KeyValuePair[string]{
			Key: "Zone", Value: g.provider.zone,
		})
	}

	return lines, nil
}

func (g *GCPHome) pathForData(key, app, stage string) string {
	return path.Join(key, app, fmt.Sprintf("%v.json", stage))
}

func (g *GCPHome) pathForPassphrase(app string, stage string) string {
	return fmt.Sprintf("sst-passphrase-%s-%s", app, stage)
}

func firstNonEmptyEnv(envs ...string) string {
	for _, env := range envs {
		if value := os.Getenv(env); value != "" {
			return value
		}
	}
	return ""
}
