package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPKUploadGrantRejectsInvalidMetadata(t *testing.T) {
	for _, body := range []string{
		`{"version":"next","sha256":"aa","size":50}`,
		`{"version":"2.1.0","sha256":"` + strings.Repeat("a", 64) + `","size":268435457}`,
		`{"version":"2.1.0","sha256":"` + strings.Repeat("z", 64) + `","size":50}`,
	} {
		response := httptest.NewRecorder()
		(&AuthDependencies{}).apkTicket(response, httptest.NewRequest("POST", "/", strings.NewReader(body)), User{})
		if response.Code != 422 {
			t.Fatalf("invalid metadata returned %d", response.Code)
		}
	}
}
func TestAndroidSettingsRejectForeignStoreLinks(t *testing.T) {
	for _, link := range []string{"https://evil.example/store", "https://play.google.com/store/apps/details?id=com.other.app", "http://play.google.com/store/apps/details?id=com.orixridesharestudents.app"} {
		response := httptest.NewRecorder()
		body := `{"support_email":"support@example.com","play_store_url":"` + link + `"}`
		(&AuthDependencies{}).setPublicSettings(response, httptest.NewRequest("PUT", "/", strings.NewReader(body)), User{})
		if response.Code != 422 {
			t.Fatalf("unsafe store link returned %d", response.Code)
		}
	}
}
