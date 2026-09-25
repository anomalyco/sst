package provider

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestNoncurrentVersionIdentifiers(t *testing.T) {
	t.Parallel()

	key := "app/my-app/production.json"
	out := &s3.ListObjectVersionsOutput{
		Versions: []s3types.ObjectVersion{
			{Key: aws.String(key), VersionId: aws.String("current"), IsLatest: aws.Bool(true)},
			{Key: aws.String(key), VersionId: aws.String("old-state"), IsLatest: aws.Bool(false)},
			{Key: aws.String("app/my-app/production.json.backup"), VersionId: aws.String("other"), IsLatest: aws.Bool(false)},
		},
		DeleteMarkers: []s3types.DeleteMarkerEntry{
			{Key: aws.String(key), VersionId: aws.String("old-marker"), IsLatest: aws.Bool(false)},
		},
	}

	got := noncurrentVersionIdentifiers(out, key)
	want := []s3types.ObjectIdentifier{
		{Key: aws.String(key), VersionId: aws.String("old-state")},
		{Key: aws.String(key), VersionId: aws.String("old-marker")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
