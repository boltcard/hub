package web

import "testing"

// Real OCI image index served by Docker Hub for boltcard/card:latest since the
// CI switched to docker/build-push-action (buildx provenance attestations on by
// default). It has a manifests[] array and NO top-level config — the shape that
// broke CheckLatestVersion (issue #45).
const ociIndexManifest = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:d9ddd6641c710953c4c6dc47a67e51b5f97b66fa1c15a2c87cd7b82349f4ab94",
      "size": 1621,
      "platform": { "architecture": "amd64", "os": "linux" }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:7ea8e9054784c94adb225247df6bfae751f20e53918742f3d9e9caaab58a3007",
      "size": 565,
      "annotations": {
        "vnd.docker.reference.digest": "sha256:d9ddd6641c710953c4c6dc47a67e51b5f97b66fa1c15a2c87cd7b82349f4ab94",
        "vnd.docker.reference.type": "attestation-manifest"
      },
      "platform": { "architecture": "unknown", "os": "unknown" }
    }
  ]
}`

// Single-platform image manifest (what the index entry points to, and what plain
// `docker push` produced before the CI change). It carries the config digest.
const plainImageManifest = `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.oci.image.config.v1+json",
    "digest": "sha256:161fa579ed83ddbfaec540cd26c8b7a5c88143dc95b44d506cb91fe6b1710d4d",
    "size": 2500
  },
  "layers": []
}`

func TestParseManifest_OCIIndexSelectsAmd64(t *testing.T) {
	configDigest, childDigest, err := parseManifest([]byte(ociIndexManifest))
	if err != nil {
		t.Fatalf("parseManifest returned error: %v", err)
	}
	if configDigest != "" {
		t.Errorf("expected no config digest for an index, got %q", configDigest)
	}
	want := "sha256:d9ddd6641c710953c4c6dc47a67e51b5f97b66fa1c15a2c87cd7b82349f4ab94"
	if childDigest != want {
		t.Errorf("expected amd64 child digest %q, got %q", want, childDigest)
	}
}

func TestParseManifest_PlainImageReturnsConfigDigest(t *testing.T) {
	configDigest, childDigest, err := parseManifest([]byte(plainImageManifest))
	if err != nil {
		t.Fatalf("parseManifest returned error: %v", err)
	}
	if childDigest != "" {
		t.Errorf("expected no child digest for a plain manifest, got %q", childDigest)
	}
	want := "sha256:161fa579ed83ddbfaec540cd26c8b7a5c88143dc95b44d506cb91fe6b1710d4d"
	if configDigest != want {
		t.Errorf("expected config digest %q, got %q", want, configDigest)
	}
}

// An index with no linux/amd64 entry should fall back to the first non-attestation
// platform manifest (e.g. an arm64-only build) rather than the attestation one.
func TestParseManifest_IndexFallsBackToNonAttestation(t *testing.T) {
	const arm64Index = `{
      "mediaType": "application/vnd.oci.image.index.v1+json",
      "manifests": [
        {
          "digest": "sha256:aaaa",
          "annotations": { "vnd.docker.reference.type": "attestation-manifest" },
          "platform": { "architecture": "unknown", "os": "unknown" }
        },
        {
          "digest": "sha256:bbbb",
          "platform": { "architecture": "arm64", "os": "linux" }
        }
      ]
    }`
	_, childDigest, err := parseManifest([]byte(arm64Index))
	if err != nil {
		t.Fatalf("parseManifest returned error: %v", err)
	}
	if childDigest != "sha256:bbbb" {
		t.Errorf("expected arm64 child digest sha256:bbbb, got %q", childDigest)
	}
}

func TestParseManifest_NoConfigErrors(t *testing.T) {
	if _, _, err := parseManifest([]byte(`{"schemaVersion":2}`)); err == nil {
		t.Error("expected error for manifest with no config and no manifests array")
	}
}
