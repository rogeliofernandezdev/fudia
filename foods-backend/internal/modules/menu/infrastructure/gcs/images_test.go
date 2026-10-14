package gcs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testSigner() *Images {
	return &Images{bucket: "test-images", signer: "api@test-project.iam.gserviceaccount.com", cache: map[string]signedImage{}, signOverride: func(context.Context, []byte) ([]byte, error) { return []byte("test-signature"), nil }}
}
func TestSignedPostPolicyBindsBucketKeyContentTypeAndRealSizeLimit(t *testing.T) {
	signer := testSigner()
	key := "restaurants/org/products/id/original.png"
	out, err := signer.Authorize(context.Background(), domain.ImageRecord{ID: "id", OriginalKey: key, ContentType: "image/png", ExpiresAt: time.Now().Add(10 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if out.Method != "POST" || out.UploadURL != "https://storage.googleapis.com/test-images/" || out.Fields["key"] != key {
		t.Fatalf("wrong policy %#v", out)
	}
	decoded, err := base64.StdEncoding.DecodeString(out.Fields["policy"])
	if err != nil {
		t.Fatal(err)
	}
	var policy struct {
		Conditions []json.RawMessage `json:"conditions"`
	}
	if err = json.Unmarshal(decoded, &policy); err != nil {
		t.Fatal(err)
	}
	content := string(decoded)
	for _, value := range []string{"content-length-range", "5242880", "image/png", "test-images", key} {
		if !strings.Contains(content, value) {
			t.Fatalf("policy missing %s", value)
		}
	}
	if strings.Contains(content, "starts-with") {
		t.Fatal("policy must bind exact object, not a wildcard")
	}
}
func TestReadURLsUseBoundedReusableSingleflightCacheAndPinnedGeneration(t *testing.T) {
	signer := testSigner()
	var calls atomic.Int32
	signer.signOverride = func(context.Context, []byte) ([]byte, error) {
		calls.Add(1)
		time.Sleep(5 * time.Millisecond)
		return []byte("signature"), nil
	}
	key := "restaurants/org/products/id/original.png"
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			url, _, err := signer.ReadURL(context.Background(), key, 42)
			if err != nil || !strings.Contains(url, "generation=42") {
				t.Errorf("signed generation: %s %v", url, err)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 1 {
		t.Fatalf("signed %d times", calls.Load())
	}
	signer.mu.Lock()
	entry := signer.cache[key+":42"]
	entry.expires = time.Now().Add(30 * time.Second)
	signer.cache[key+":42"] = entry
	signer.mu.Unlock()
	if _, _, err := signer.ReadURL(context.Background(), key, 42); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal("did not refresh before expiration")
	}
	for i := 0; i < 4096; i++ {
		signer.cache[string(rune(i))+":0"] = signedImage{"cached", time.Now().Add(time.Hour)}
	}
	if _, _, err := signer.ReadURL(context.Background(), "new-key", 0); err != nil {
		t.Fatal(err)
	}
	if len(signer.cache) > 4097 {
		t.Fatal("cache unbounded")
	}
}
