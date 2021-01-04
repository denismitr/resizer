package mgoregistry

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"resizer/media"
)

func mapSliceToMongoRecord(slice *media.Slice, sliceID primitive.ObjectID) *sliceRecord {
	if slice.ID.None() && sliceID.IsZero() {
		panic("how can both media ID and mongo ID be empty")
	}

	imgID, err := primitive.ObjectIDFromHex(slice.ImageID.String())
	if err != nil {
		panic(err)
	}

	sr := sliceRecord{
		ImageID:   imgID,
		Name:      slice.Name,
		Bucket:    slice.Bucket,
		Width:     slice.Width,
		Height:    slice.Height,
		Format:    slice.Format,
		CreatedAt: slice.CreatedAt,
		Size:      slice.Size,
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
		ID:        media.ID(sr.ID.Hex()),
		ImageID:   media.ID(sr.ImageID.Hex()),
		Name:      sr.Name,
		Format:    sr.Format,
		Bucket:    sr.Bucket,
		Width:     sr.Width,
		Height:    sr.Height,
		CreatedAt: sr.CreatedAt,
		IsValid:   sr.IsValid,
		Size:      sr.Size,
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
		Path:         ir.Path,
		Url:          ir.Url,
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
		Path:         img.Path,
		Url:          img.Url,
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
