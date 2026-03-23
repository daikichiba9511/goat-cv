package usecase

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/chibadaimare/goat/backend/internal/domain"
	"github.com/chibadaimare/goat/backend/internal/repository/sqlite"
	"github.com/google/uuid"
)

type ImageUsecase struct {
	repo        *sqlite.ImageRepository
	storagePath string
}

func NewImageUsecase(repo *sqlite.ImageRepository, storagePath string) *ImageUsecase {
	return &ImageUsecase{repo: repo, storagePath: storagePath}
}

func (u *ImageUsecase) Upload(ctx context.Context, projectID, filename string, file io.Reader) (domain.Image, error) {
	id := uuid.Must(uuid.NewV7()).String()

	localPath := filepath.Join(u.storagePath, id+filepath.Ext(filename))
	f, err := os.Create(localPath)
	if err != nil {
		return domain.Image{}, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, file); err != nil {
		return domain.Image{}, fmt.Errorf("save file: %w", err)
	}

	w, h, err := getImageDimensions(localPath)
	if err != nil {
		os.Remove(localPath)
		return domain.Image{}, fmt.Errorf("read image dimensions: %w", err)
	}

	img := domain.Image{
		ID:             id,
		ProjectID:      projectID,
		Filename:       filename,
		OriginalWidth:  w,
		OriginalHeight: h,
		Width:          w,
		Height:         h,
		Rotation:       domain.Rotation0,
		FlipH:          false,
		FlipV:          false,
		Status:         domain.ImageStatusPending,
	}
	return u.repo.Create(ctx, img)
}

func (u *ImageUsecase) Get(ctx context.Context, id string) (domain.Image, error) {
	return u.repo.Get(ctx, id)
}

func (u *ImageUsecase) ListByProject(ctx context.Context, projectID string) ([]domain.Image, error) {
	return u.repo.ListByProject(ctx, projectID)
}

func (u *ImageUsecase) FilePath(img domain.Image) string {
	return filepath.Join(u.storagePath, img.ID+filepath.Ext(img.Filename))
}

func (u *ImageUsecase) UpdateTransform(ctx context.Context, id string, rotation domain.Rotation, flipH, flipV bool) (domain.Image, error) {
	img, err := u.repo.Get(ctx, id)
	if err != nil {
		return domain.Image{}, err
	}
	w, h := domain.EffectiveDimensions(img.OriginalWidth, img.OriginalHeight, rotation)
	return u.repo.UpdateTransform(ctx, id, rotation, flipH, flipV, w, h)
}

func (u *ImageUsecase) Delete(ctx context.Context, id string) error {
	img, err := u.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	os.Remove(u.FilePath(img))
	return u.repo.Delete(ctx, id)
}

func getImageDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
