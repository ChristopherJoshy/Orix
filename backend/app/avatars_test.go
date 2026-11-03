package server

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

type onboardingAvatarStore struct {
	fakeUsers
	owner string
	data  []byte
}

func (s *onboardingAvatarStore) SaveAvatar(_ context.Context, owner string, data []byte) (User, error) {
	s.owner = owner
	s.data = data
	return User{ID: owner, Onboarded: false, Status: "active"}, nil
}
func (s *onboardingAvatarStore) ReadAvatar(context.Context, string, string) (Avatar, error) {
	return Avatar{}, nil
}
func TestOnboardingPhotoRequiresAuthenticationButNotCompletedOnboarding(t *testing.T) {
	var photo bytes.Buffer
	if err := png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, token, status string
		want                int
	}{
		{"new verified account", "Bearer valid", "active", 200},
		{"anonymous", "", "active", 401},
		{"restricted", "Bearer valid", "banned", 403},
	} {
		t.Run(test.name, func(t *testing.T) {
			var payload bytes.Buffer
			writer := multipart.NewWriter(&payload)
			part, err := writer.CreateFormFile("file", "face.png")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = part.Write(photo.Bytes()); err != nil {
				t.Fatal(err)
			}
			writer.Close()
			store := &onboardingAvatarStore{fakeUsers: fakeUsers{status: test.status}}
			handler := New(Config{Auth: &AuthDependencies{fakeVerifier{}, store}}, nil)
			request := httptest.NewRequest("POST", "/api/v1/users/me/avatar", &payload)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			request.Header.Set("Authorization", test.token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status %d, body %s", response.Code, response.Body.String())
			}
			if test.want == 200 && (store.owner != "u" || len(store.data) == 0) {
				t.Fatal("authenticated photo was not stored")
			}
			if test.want != 200 && store.owner != "" {
				t.Fatal("unauthorized upload reached storage")
			}
		})
	}
}
