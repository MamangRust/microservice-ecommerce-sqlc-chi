package handler

import (
	"encoding/json"
	"log"

	db "github.com/MamangRust/microservice-ecommerce-grpc-category/database/schema"
	"github.com/MamangRust/microservice-ecommerce-grpc-pb/category"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (h *Handler) mapToCategoryResponse(data interface{}) interface{} {
	switch v := data.(type) {
	case *db.GetCategoryByIDRow:
		return &pb_category.CategoryResponse{
			Id:            int32(v.CategoryID),
			Name:          v.Name,
			Description:   derefString(v.Description),
			SlugCategory:  derefString(v.SlugCategory),
			ImageCategory: derefString(v.ImageCategory),
			CreatedAt:     v.CreatedAt.Time.Format("2006-01-02"),
			UpdatedAt:     v.UpdatedAt.Time.Format("2006-01-02"),
		}
	case *db.GetCategoriesRow:
		return &pb_category.CategoryResponse{
			Id:            int32(v.CategoryID),
			Name:          v.Name,
			Description:   derefString(v.Description),
			SlugCategory:  derefString(v.SlugCategory),
			ImageCategory: derefString(v.ImageCategory),
			CreatedAt:     v.CreatedAt.Time.Format("2006-01-02"),
			UpdatedAt:     v.UpdatedAt.Time.Format("2006-01-02"),
		}
	case *db.GetCategoriesActiveRow:
		var deletedAt string
		if v.DeletedAt.Valid {
			deletedAt = v.DeletedAt.Time.Format("2006-01-02")
		}
		return &pb_category.CategoryResponseDeleteAt{
			Id:            int32(v.CategoryID),
			Name:          v.Name,
			Description:   derefString(v.Description),
			SlugCategory:  derefString(v.SlugCategory),
			ImageCategory: derefString(v.ImageCategory),
			CreatedAt:     v.CreatedAt.Time.Format("2006-01-02"),
			UpdatedAt:     v.UpdatedAt.Time.Format("2006-01-02"),
			DeletedAt:     &wrapperspb.StringValue{Value: deletedAt},
		}
	case *db.GetCategoriesTrashedRow:
		var deletedAt string
		if v.DeletedAt.Valid {
			deletedAt = v.DeletedAt.Time.Format("2006-01-02")
		}
		return &pb_category.CategoryResponseDeleteAt{
			Id:            int32(v.CategoryID),
			Name:          v.Name,
			Description:   derefString(v.Description),
			SlugCategory:  derefString(v.SlugCategory),
			ImageCategory: derefString(v.ImageCategory),
			CreatedAt:     v.CreatedAt.Time.Format("2006-01-02"),
			UpdatedAt:     v.UpdatedAt.Time.Format("2006-01-02"),
			DeletedAt:     &wrapperspb.StringValue{Value: deletedAt},
		}
	case *db.CreateCategoryRow:
		return &pb_category.CategoryResponse{
			Id:            int32(v.CategoryID),
			Name:          v.Name,
			Description:   derefString(v.Description),
			SlugCategory:  derefString(v.SlugCategory),
			ImageCategory: derefString(v.ImageCategory),
			CreatedAt:     v.CreatedAt.Time.Format("2006-01-02"),
			UpdatedAt:     v.UpdatedAt.Time.Format("2006-01-02"),
		}
	case *db.UpdateCategoryRow:
		return &pb_category.CategoryResponse{
			Id:            int32(v.CategoryID),
			Name:          v.Name,
			Description:   derefString(v.Description),
			SlugCategory:  derefString(v.SlugCategory),
			ImageCategory: derefString(v.ImageCategory),
			CreatedAt:     v.CreatedAt.Time.Format("2006-01-02"),
			UpdatedAt:     v.UpdatedAt.Time.Format("2006-01-02"),
		}
	case *db.Category:
		var deletedAt string
		if v.DeletedAt.Valid {
			deletedAt = v.DeletedAt.Time.Format("2006-01-02")
		}
		return &pb_category.CategoryResponseDeleteAt{
			Id:            int32(v.CategoryID),
			Name:          v.Name,
			Description:   derefString(v.Description),
			SlugCategory:  derefString(v.SlugCategory),
			ImageCategory: derefString(v.ImageCategory),
			CreatedAt:     v.CreatedAt.Time.Format("2006-01-02"),
			UpdatedAt:     v.UpdatedAt.Time.Format("2006-01-02"),
			DeletedAt:     &wrapperspb.StringValue{Value: deletedAt},
		}
	default:
		log.Printf("Unknown type for mapping: %T", v)
		return nil
	}
}

// derefString reads an optional text column. The category table allows
// description/slug_category/image_category to be NULL, and a nil deref here
// would take the whole gRPC server down, so NULLs map to the empty string.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *Handler) mapToPayload(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
