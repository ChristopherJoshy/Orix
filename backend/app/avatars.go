package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	_ "golang.org/x/image/webp"
)

type Avatar struct {
	UserID      string `bson:"user_id"`
	Data        []byte `bson:"image_data"`
	ContentType string `bson:"content_type"`
	Digest      string `bson:"digest"`
}
type AvatarRepository interface {
	SaveAvatar(context.Context, string, []byte) (User, error)
	ReadAvatar(context.Context, string, string) (Avatar, error)
}

var avatarSlots = make(chan struct{}, 4)
var ErrInvalidAvatar = errors.New("use a JPEG, PNG or WebP image up to 2 MB and 4096 pixels per side")

func normalizeAvatar(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > 2<<20 {
		return nil, ErrInvalidAvatar
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png" && format != "webp") || config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || config.Width*config.Height > 16000000 {
		return nil, ErrInvalidAvatar
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalidAvatar
	}
	width, height := config.Width, config.Height
	if width > 512 || height > 512 {
		if width >= height {
			height = max(1, height*512/width)
			width = 512
		} else {
			width = max(1, width*512/height)
			height = 512
		}
	}
	thumbnail := image.NewRGBA(image.Rect(0, 0, width, height))
	bounds := decoded.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			red, green, blue, alpha := decoded.At(bounds.Min.X+x*bounds.Dx()/width, bounds.Min.Y+y*bounds.Dy()/height).RGBA()
			thumbnail.SetRGBA(x, y, color.RGBA{uint8((red + 65535 - alpha) >> 8), uint8((green + 65535 - alpha) >> 8), uint8((blue + 65535 - alpha) >> 8), 255})
		}
	}
	var output bytes.Buffer
	if err = jpeg.Encode(&output, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return nil, ErrInvalidAvatar
	}
	return output.Bytes(), nil
}
func (m mongoUsers) SaveAvatar(ctx context.Context, user string, data []byte) (User, error) {
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	photo := "/api/v1/users/" + url.PathEscape(user) + "/avatar?v=" + digest[:16]
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return User{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		var current User
		if err := m.collection.FindOne(tx, bson.M{"user_id": user}).Decode(&current); err != nil {
			return nil, err
		}
		if accountRestricted(current) {
			return nil, ErrGroupUnavailable
		}
		avatar := Avatar{user, data, "image/jpeg", digest}
		if _, err := m.collection.Database().Collection("user_avatars").UpdateOne(tx, bson.M{"user_id": user}, bson.M{"$set": avatar}, options.UpdateOne().SetUpsert(true)); err != nil {
			return nil, err
		}
		var updated User
		err := m.collection.FindOneAndUpdate(tx, bson.M{"user_id": user}, bson.M{"$set": bson.M{"photo_url": photo, "updated_at": time.Now().UTC()}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
		return updated, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return User{}, err
	}
	userRecord, ok := result.(User)
	if !ok {
		return User{}, errors.New("missing profile result")
	}
	return userRecord, nil
}
func (m mongoUsers) ReadAvatar(ctx context.Context, viewer, target string) (Avatar, error) {
	if viewer != target {
		blocked, err := m.BlockedPeers(ctx, viewer)
		if err != nil {
			return Avatar{}, err
		}
		if blocked[target] {
			return Avatar{}, mongo.ErrNoDocuments
		}
		if err := m.collection.Database().Collection("ride_groups").FindOne(ctx, bson.M{"members.user_id": bson.M{"$all": []string{viewer, target}}, "status": bson.M{"$in": []string{"pending", "confirming", "active", "in_progress", "completed"}}}).Err(); err != nil {
			return Avatar{}, err
		}
	}
	var profile User
	if err := m.collection.FindOne(ctx, bson.M{"user_id": target}).Decode(&profile); err != nil {
		return Avatar{}, err
	}
	if accountRestricted(profile) {
		return Avatar{}, mongo.ErrNoDocuments
	}
	var avatar Avatar
	err := m.collection.Database().Collection("user_avatars").FindOne(ctx, bson.M{"user_id": target}).Decode(&avatar)
	if err == nil && avatar.Digest == "" {
		sum := sha256.Sum256(avatar.Data)
		avatar.Digest = hex.EncodeToString(sum[:])
	}
	return avatar, err
}
func (a *AuthDependencies) uploadAvatar(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.Users.(AvatarRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Avatar storage unavailable"})
		return
	}
	select {
	case avatarSlots <- struct{}{}:
		defer func() { <-avatarSlots }()
	default:
		w.Header().Set("Retry-After", "1")
		writeJSON(w, 503, map[string]string{"detail": "Image processing is busy. Please retry."})
		return
	}
	reader, err := r.MultipartReader()
	if err != nil {
		writeJSON(w, 422, map[string]string{"detail": "Upload a multipart image in the file field"})
		return
	}
	var data []byte
	for {
		part, partErr := reader.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			writeJSON(w, 422, map[string]string{"detail": "Invalid upload"})
			return
		}
		if part.FormName() != "file" || data != nil {
			part.Close()
			writeJSON(w, 422, map[string]string{"detail": "Upload exactly one image"})
			return
		}
		data, err = io.ReadAll(io.LimitReader(part, (2<<20)+1))
		part.Close()
		if err != nil || len(data) > 2<<20 {
			writeJSON(w, 413, map[string]string{"detail": "Image exceeds 2 MB"})
			return
		}
	}
	normalized, err := normalizeAvatar(data)
	if err != nil {
		writeJSON(w, 422, map[string]string{"detail": ErrInvalidAvatar.Error()})
		return
	}
	updated, err := repository.SaveAvatar(r.Context(), user.ID, normalized)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to save profile image"})
		return
	}
	writeJSON(w, 200, profile(updated))
}
func (a *AuthDependencies) getAvatar(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.Users.(AvatarRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Avatar storage unavailable"})
		return
	}
	avatar, err := repository.ReadAvatar(r.Context(), user.ID, r.PathValue("user_id"))
	if errors.Is(err, mongo.ErrNoDocuments) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load image"})
		return
	}
	etag := "\"" + avatar.Digest + "\""
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", avatar.ContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(avatar.Data)
}
