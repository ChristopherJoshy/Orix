package main

import (
	"strings"
	"testing"
)

func TestReleaseOperatorSelectionGuards(t *testing.T) {
	t.Setenv("RELEASE_APPLY", "true")
	t.Setenv("RELEASE_DATABASE", "release-test")
	t.Setenv("APP_ENV", "production")
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	t.Setenv("ADMIN_API_SECRET", strings.Repeat("x", 32))
	digest := strings.Repeat("a", 64)
	if err := validateReleaseSelection("release-test", "/tmp/release.apk", "2.1.2", digest); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ key, value string }{{"RELEASE_APPLY", "false"}, {"RELEASE_DATABASE", "other"}, {"APP_ENV", "local"}, {"FIREBASE_AUTH_EMULATOR_HOST", "localhost:9099"}, {"ADMIN_API_SECRET", ""}} {
		t.Run(test.key, func(t *testing.T) {
			t.Setenv(test.key, test.value)
			if validateReleaseSelection("release-test", "/tmp/release.apk", "2.1.2", digest) == nil {
				t.Fatal("missing operator guard")
			}
		})
	}
	if validateReleaseSelection("release-test", "/tmp/release.apk", "latest", digest) == nil {
		t.Fatal("non-version accepted")
	}
	if validateReleaseSelection("release-test", "/tmp/release.apk", "2.1.2", "bad") == nil {
		t.Fatal("invalid digest accepted")
	}
}
