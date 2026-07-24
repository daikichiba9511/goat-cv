package usecase

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/google/uuid"
)

type imageRepository interface {
	Create(ctx context.Context, image domain.Image) (domain.Image, error)
	Get(ctx context.Context, id string) (domain.Image, error)
	ListByProject(ctx context.Context, projectID string) ([]domain.Image, error)
	ListByProjectFiltered(ctx context.Context, projectID string, status *domain.ImageStatus, escalated *bool) ([]domain.Image, error)
	UpdateTransform(ctx context.Context, id string, rotation domain.Rotation, flipH, flipV bool, width, height int) (domain.Image, error)
	UpdateWorkflow(ctx context.Context, id string, status domain.ImageStatus, escalated bool) (domain.Image, error)
	Delete(ctx context.Context, id string) error
}

// ErrInvalidImageStatus indicates a lifecycle status outside the workflow contract.
var ErrInvalidImageStatus = errors.New("invalid image status")

// ImageListFilter contains optional lifecycle and escalation filters combined with AND semantics.
type ImageListFilter struct {
	Status    *domain.ImageStatus
	Escalated *bool
}

// ImageUsecase coordinates image file storage and image metadata operations.
type ImageUsecase struct {
	repo        imageRepository
	storagePath string
}

// NewImageUsecase creates an ImageUsecase.
func NewImageUsecase(repo imageRepository, storagePath string) *ImageUsecase {
	return &ImageUsecase{repo: repo, storagePath: storagePath}
}

// Upload stores an image file and creates its metadata record.
// The returned Image keeps the original filename for display, while the local file path is derived from the generated image ID.
func (u *ImageUsecase) Upload(ctx context.Context, projectID, filename string, file io.Reader) (domain.Image, error) {
	id := uuid.Must(uuid.NewV7()).String()

	// Why: 保存名をUUIDにして、同名アップロードの衝突とユーザー入力由来パスを避ける。
	storageFilePath := filepath.Join(u.storagePath, id+filepath.Ext(filename))
	storedFile, err := os.Create(storageFilePath)
	if err != nil {
		return domain.Image{}, fmt.Errorf("create file: %w", err)
	}
	defer storedFile.Close()

	if _, err := io.Copy(storedFile, file); err != nil {
		return domain.Image{}, fmt.Errorf("save file: %w", err)
	}

	width, height, err := getImageDimensions(storageFilePath)
	if err != nil {
		// Why not: 画像として読めないファイルはDBに残さない。ファイル保存だけ成功した状態を作らない。
		os.Remove(storageFilePath)
		return domain.Image{}, fmt.Errorf("read image dimensions: %w", err)
	}

	img := domain.Image{
		ID:             id,
		ProjectID:      projectID,
		Filename:       filename,
		OriginalWidth:  width,
		OriginalHeight: height,
		Width:          width,
		Height:         height,
		Rotation:       domain.Rotation0,
		FlipH:          false,
		FlipV:          false,
		Status:         domain.ImageStatusPending,
		Escalated:      false,
	}
	return u.repo.Create(ctx, img)
}

// Get returns image metadata by ID.
func (u *ImageUsecase) Get(ctx context.Context, id string) (domain.Image, error) {
	return u.repo.Get(ctx, id)
}

// ListByProject returns images for a project.
func (u *ImageUsecase) ListByProject(ctx context.Context, projectID string) ([]domain.Image, error) {
	return u.repo.ListByProject(ctx, projectID)
}

// ListByProjectFiltered returns Project Images matching all provided workflow filters.
func (u *ImageUsecase) ListByProjectFiltered(
	ctx context.Context,
	projectID string,
	filter ImageListFilter,
) ([]domain.Image, error) {
	if filter.Status != nil && !isKnownImageStatus(*filter.Status) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidImageStatus, *filter.Status)
	}
	return u.repo.ListByProjectFiltered(ctx, projectID, filter.Status, filter.Escalated)
}

// FilePath returns the local storage path for an image.
// The path is derived from the image ID and original extension; callers should not persist this value.
func (u *ImageUsecase) FilePath(img domain.Image) string {
	// Why: DBには元ファイル名を残し、ローカル保存場所はIDと拡張子から再構成する。
	return filepath.Join(u.storagePath, img.ID+filepath.Ext(img.Filename))
}

// UpdateTransform updates image transform metadata and effective dimensions.
// It does not rewrite the stored image file.
func (u *ImageUsecase) UpdateTransform(ctx context.Context, id string, rotation domain.Rotation, flipH, flipV bool) (domain.Image, error) {
	img, err := u.repo.Get(ctx, id)
	if err != nil {
		return domain.Image{}, err
	}
	if err := requireImageWorkflowOperation(img, ImageWorkflowOperationTransformEdit); err != nil {
		return domain.Image{}, err
	}
	// Why: 回転・反転は画像ファイルを書き換えず、annotatorが見る座標空間のサイズだけを更新する。
	width, height := domain.EffectiveDimensions(img.OriginalWidth, img.OriginalHeight, rotation)
	return u.repo.UpdateTransform(ctx, id, rotation, flipH, flipV, width, height)
}

// Delete removes image metadata and its local file.
// Missing local files are ignored so stale storage does not block metadata cleanup.
func (u *ImageUsecase) Delete(ctx context.Context, id string) error {
	img, err := u.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	os.Remove(u.FilePath(img))
	return u.repo.Delete(ctx, id)
}

func getImageDimensions(path string) (int, int, error) {
	imageFile, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer imageFile.Close()

	config, _, err := image.DecodeConfig(imageFile)
	if err != nil {
		return 0, 0, err
	}
	return config.Width, config.Height, nil
}
