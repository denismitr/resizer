package mgoregistry

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"resizer/media"
	"time"
)

type imageRecord struct {
	ID           primitive.ObjectID `bson:"_id"`
	Name         string             `bson:"name"`
	OriginalName string             `bson:"originalName"`
	OriginalSize int                `bson:"originalSize"`
	OriginalExt  string             `bson:"originalExt"`
	PublishAt    *time.Time         `bson:"publishAt"`
	CreatedAt    time.Time          `bson:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"`
	Namespace    string             `bson:"namespace"`
}

type sliceRecord struct {
	ID         primitive.ObjectID `bson:"_id"`
	ImageID    primitive.ObjectID `bson:"imageId"`
	Filename   string             `bson:"filename"`
	Namespace  string             `bson:"namespace"`
	Extension  string             `bson:"extension"`
	Cropped    bool               `bson:"cropped"`
	Path       string             `bson:"path"`
	Width      int                `bson:"width"`
	Height     int                `bson:"height"`
	Size       int                `bson:"size"`
	Quality    int                `bson:"quality"`
	Mime       string             `json:"mime"`
	CreatedAt  time.Time          `bson:"createdAt"`
	IsValid    bool               `bson:"isValid"`
	IsOriginal bool               `bson:"isOriginal"`
	Status     string             `bson:"status"`
}

func mapSliceToMongoRecord(slice *media.Slice) *sliceRecord {
	if slice.ID.None() {
		panic("how can both slice ID be empty")
	}

	sliceID, err := primitive.ObjectIDFromHex(slice.ID.String())
	if err != nil {
		panic(fmt.Sprintf("invalid slice ID [%s]", slice.ID.String()))
	}

	imgID, err := primitive.ObjectIDFromHex(slice.ImageID.String())
	if err != nil {
		panic(fmt.Sprintf("invalid slice image ID [%s]", slice.ImageID.String()))
	}

	return &sliceRecord{
		ID:         sliceID,
		ImageID:    imgID,
		Filename:   slice.Filename,
		Namespace:  slice.Namespace,
		Cropped:    slice.Cropped,
		Path:       slice.Path,
		Width:      slice.Width,
		Height:     slice.Height,
		Quality:    slice.Quality,
		Mime:       slice.Mime,
		Extension:  slice.Extension,
		CreatedAt:  slice.CreatedAt,
		Size:       slice.Size,
		IsOriginal: slice.IsOriginal,
		IsValid:    slice.IsValid,
		Status:     string(slice.Status),
	}
}

func mapMongoRecordToSlice(sr *sliceRecord) *media.Slice {
	return &media.Slice{
		ID:         media.ID(sr.ID.Hex()),
		ImageID:    media.ID(sr.ImageID.Hex()),
		Filename:   sr.Filename,
		Extension:  sr.Extension,
		Namespace:  sr.Namespace,
		Cropped:    sr.Cropped,
		Width:      sr.Width,
		Height:     sr.Height,
		Quality:    sr.Quality,
		Mime:       sr.Mime,
		Path:       sr.Path,
		CreatedAt:  sr.CreatedAt,
		IsValid:    sr.IsValid,
		IsOriginal: sr.IsOriginal,
		Size:       sr.Size,
		Status:     media.Status(sr.Status),
	}
}

func mapMongoRecordToImage(ir *imageRecord) *media.Image {
	return &media.Image{
		ID:           media.ID(ir.ID.Hex()),
		Name:         ir.Name,
		OriginalName: ir.OriginalName,
		OriginalExt:  ir.OriginalExt,
		OriginalSize: ir.OriginalSize,
		Namespace:    ir.Namespace,
		CreatedAt:    ir.CreatedAt,
		UpdatedAt:    ir.UpdatedAt,
		PublishAt:    ir.PublishAt,
	}
}

func mapImageToMongoRecord(img *media.Image) *imageRecord {
	if img.ID.None() {
		panic("how can image ID be empty")
	}

	imgID, err := primitive.ObjectIDFromHex(img.ID.String())
	if err != nil {
		panic(fmt.Sprintf("invalid image ID [%s]", img.ID.String()))
	}

	return &imageRecord{
		ID:           imgID,
		Name:         img.Name,
		OriginalName: img.OriginalName,
		OriginalSize: img.OriginalSize,
		OriginalExt:  img.OriginalExt,
		Namespace:    img.Namespace,
		CreatedAt:    img.CreatedAt,
		UpdatedAt:    img.UpdatedAt,
		PublishAt:    img.PublishAt,
	}
}
