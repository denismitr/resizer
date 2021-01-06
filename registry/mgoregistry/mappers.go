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
	PublishAt    *time.Time         `bson:"publishedAt"`
	CreatedAt    time.Time          `bson:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"`
	Bucket       string             `bson:"bucket"`
}

type sliceRecord struct {
	ID         primitive.ObjectID `bson:"_id"`
	ImageID    primitive.ObjectID `bson:"imageId"`
	Filename   string             `bson:"filename"`
	Bucket     string             `bson:"bucket"`
	Format     string             `bson:"format"`
	Width      int                `bson:"width"`
	Height     int                `bson:"height"`
	Size       int                `bson:"size"`
	CreatedAt  time.Time          `bson:"createdAt"`
	IsValid    bool               `bson:"isValid"`
	IsOriginal bool               `bson:"isOriginal"`
}

func mapSliceToMongoRecord(slice *media.Slice, sliceID primitive.ObjectID) *sliceRecord {
	if slice.ID.None() && sliceID.IsZero() {
		panic("how can both media ID and mongo ID be empty")
	}

	imgID, err := primitive.ObjectIDFromHex(slice.ImageID.String())
	if err != nil {
		panic(fmt.Sprintf("invalid slice image ID [%s]", slice.ImageID.String()))
	}

	sr := sliceRecord{
		ImageID:   imgID,
		Filename:  slice.Filename,
		Bucket:    slice.Bucket,
		Width:     slice.Width,
		Height:    slice.Height,
		Format:    slice.Extension,
		CreatedAt: slice.CreatedAt,
		Size:      slice.Size,
		IsOriginal: slice.IsOriginal,
		IsValid: slice.IsValid,
	}

	if slice.ID.None() {
		sr.ID = sliceID
	} else {
		if ID, err := primitive.ObjectIDFromHex(slice.ID.String()); err != nil {
			panic("how can slice ID be invalid")
		} else {
			sr.ID = ID
		}
	}

	return &sr
}

func mapMongoRecordToSlice(sr *sliceRecord) *media.Slice {
	return &media.Slice{
		ID:         media.ID(sr.ID.Hex()),
		ImageID:    media.ID(sr.ImageID.Hex()),
		Filename:   sr.Filename,
		Extension:  sr.Format,
		Bucket:     sr.Bucket,
		Width:      sr.Width,
		Height:     sr.Height,
		CreatedAt:  sr.CreatedAt,
		IsValid:    sr.IsValid,
		IsOriginal: sr.IsOriginal,
		Size:       sr.Size,
	}
}

func mapMongoRecordToImage(ir *imageRecord) *media.Image {
	return &media.Image{
		ID:           media.ID(ir.ID.Hex()),
		Name:         ir.Name,
		OriginalName: ir.OriginalName,
		OriginalExt:  ir.OriginalExt,
		OriginalSize: ir.OriginalSize,
		Bucket:       ir.Bucket,
		CreatedAt:    ir.CreatedAt,
		UpdatedAt:    ir.UpdatedAt,
		PublishAt:    ir.PublishAt,
	}
}

func mapImageToMongoRecord(img *media.Image, mongoID primitive.ObjectID) *imageRecord {
	if img.ID.None() && mongoID.IsZero() {
		panic("how can both media ID and mongo ID be empty")
	}

	ir := imageRecord{
		Name:         img.Name,
		OriginalName: img.OriginalName,
		OriginalSize: img.OriginalSize,
		OriginalExt:  img.OriginalExt,
		Bucket:       img.Bucket,
		CreatedAt:    img.CreatedAt,
		UpdatedAt:    img.UpdatedAt,
		PublishAt:    img.PublishAt,
	}

	if img.ID.None() {
		ir.ID = mongoID
	} else {
		if ID, err := primitive.ObjectIDFromHex(img.ID.String()); err != nil {
			panic("how can image ID be invalid")
		} else {
			ir.ID = ID
		}
	}

	return &ir
}
