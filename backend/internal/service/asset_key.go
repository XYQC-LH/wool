package service

import (
	"path"
	"regexp"
	"strings"
	"time"

	"nexus-api/internal/model"

	"github.com/google/uuid"
)

var safeExtRe = regexp.MustCompile(`^[a-z0-9]{1,16}$`)

func buildDatePath(now time.Time) string {
	return now.Format("2006/01/02")
}

func sanitizeExt(originalFilename string) string {
	filename := path.Base(strings.ReplaceAll(originalFilename, "\\", "/"))
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(filename)), ".")
	if ext == "" {
		return ""
	}
	if !safeExtRe.MatchString(ext) {
		return ""
	}
	return "." + ext
}

func buildObjectKey(prefix string, kind model.AssetKind, originalFilename string, now time.Time) string {
	ext := sanitizeExt(originalFilename)
	name := uuid.New().String() + ext
	return path.Join(prefix, string(kind), buildDatePath(now), name)
}

// BuildSiteObjectKey 网站素材Key：site/{kind}/yyyy/mm/dd/{uuid}{ext}
func BuildSiteObjectKey(kind model.AssetKind, originalFilename string, now time.Time) string {
	return buildObjectKey("site", kind, originalFilename, now)
}

// BuildUserUploadObjectKey 用户上传Key：users/{user_id}/uploads/{kind}/yyyy/mm/dd/{uuid}{ext}
func BuildUserUploadObjectKey(userID uuid.UUID, kind model.AssetKind, originalFilename string, now time.Time) string {
	return buildObjectKey(path.Join("users", userID.String(), "uploads"), kind, originalFilename, now)
}

// BuildUserOutputObjectKey API产物Key：users/{user_id}/outputs/{kind}/yyyy/mm/dd/{uuid}{ext}
func BuildUserOutputObjectKey(userID uuid.UUID, kind model.AssetKind, originalFilename string, now time.Time) string {
	return buildObjectKey(path.Join("users", userID.String(), "outputs"), kind, originalFilename, now)
}
