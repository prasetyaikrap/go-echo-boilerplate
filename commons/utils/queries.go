package utils

import (
	"encoding/json"
	"fmt"
	"go-serviceboilerplate/commons/models"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func GeneratePreloadRelations(db *gorm.DB, preloads *map[string][]string) *gorm.DB {
	if preloads == nil {
		return db
	}

	for relation, conditions := range *preloads {
		if len(conditions) > 0 {
			relationConditions := []any{}
			for _, condition := range conditions {
				switch {
				case strings.HasPrefix(condition, "LIMIT"):
					limitNumber, err := strconv.Atoi(strings.TrimPrefix(condition, "LIMIT "))
					if err == nil {
						relationConditions = append(relationConditions, func(db *gorm.DB) *gorm.DB {
							return db.Limit(limitNumber)
						})
					}
				default: 
					relationConditions = append(relationConditions, condition)
				}
			}
			db = db.Preload(relation, relationConditions...)
		} else {
			db = db.Preload(relation)
		}
	}

	return db
}

func GenerateQueries(db *gorm.DB, queries *models.Queries, defaultLimit int64) (*gorm.DB) {
	if queries == nil {
		return db
	}

	for _, filter := range queries.Filters {
		switch filter.Operator {
			case models.OpEqual:
				db = db.Where(fmt.Sprintf("%s = ?", filter.Column), filter.Value)
			case models.OpNotEqual:
				db = db.Where(fmt.Sprintf("%s <> ?", filter.Column), filter.Value)
			case models.OpIlike:
				db = db.Where(fmt.Sprintf("%s ILIKE ?", filter.Column), fmt.Sprintf("%%%v%%", filter.Value))
			case models.OpLike:
				db = db.Where(fmt.Sprintf("%s LIKE ?", filter.Column), fmt.Sprintf("%%%v%%", filter.Value))
			case models.OpStartsWith:
				db = db.Where(fmt.Sprintf("%s LIKE ?", filter.Column), fmt.Sprintf("%v%%", filter.Value))
			case models.OpEndsWith:
				db = db.Where(fmt.Sprintf("%s LIKE ?", filter.Column), fmt.Sprintf("%%%v", filter.Value))
			case models.OpSliceIn:
				if values, ok := filter.Value.([]any); ok {
					db = db.Where(fmt.Sprintf("%s IN ?", filter.Column), values)
				}
			case models.OpSliceNotIn:
				if values, ok := filter.Value.([]any); ok {
					db = db.Where(fmt.Sprintf("%s NOT IN ?", filter.Column), values)
				}
			case models.OpGreaterThan:
				db = db.Where(fmt.Sprintf("%s > ?", filter.Column), filter.Value)
			case models.OpGreaterThanOrEqualTo:
				db = db.Where(fmt.Sprintf("%s >= ?", filter.Column), filter.Value)
			case models.OpLessThan:
				db = db.Where(fmt.Sprintf("%s < ?", filter.Column), filter.Value)
			case models.OpLessThanOrEqualTo:
				db = db.Where(fmt.Sprintf("%s <= ?", filter.Column), filter.Value)
			case models.OpBetween:
				if values, ok := filter.Value.([]any); ok {
					if len(values) == 2 {
						db = db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", filter.Column), values[0], values[1])
					}
				}
			case models.OpNotBetween:
				if values, ok := filter.Value.([]any); ok {
					if len(values) == 2 {
						db = db.Where(fmt.Sprintf("%s NOT BETWEEN ? AND ?", filter.Column), values[0], values[1])
					}
				}
			case models.OpIsNull:
				db = db.Where(fmt.Sprintf("%s IS NULL", filter.Column))
			case models.OpIsNotNull:
				db = db.Where(fmt.Sprintf("%s IS NOT NULL", filter.Column))
			case models.OpBoolean:
				db = db.Where(fmt.Sprintf("%s = ?", filter.Column), filter.Value)
			case models.OpOr:
				columns := strings.Split(filter.Column, ";")
				var conditions []string
				var values []any
				for _, col := range columns {
					conditions = append(conditions, fmt.Sprintf("%s = ?", col))
					values = append(values, filter.Value)
				}
				db = db.Where(strings.Join(conditions, " OR "), values...)
			default:
				// unsupported operator, skip
		}
	}

	// handle default limit
	if queries.Limit <= 0 && defaultLimit > 0 {
		queries.Limit = defaultLimit
	}

	// handle default page
	if queries.Page <= 0 {
		queries.Page = 1
	}

	if queries.Limit > 0 {
		db = db.Limit(int(queries.Limit))
	}
	if queries.Page > 0 {
		offset := (queries.Page - 1) * queries.Limit
		db = db.Offset(int(offset))
	}
	if len(queries.Sorters) > 0 {
		for _, sortField := range queries.Sorters {
			if sortField == "" {
				continue
			}
			if strings.HasPrefix(sortField, "-") {
				db = db.Order(fmt.Sprintf("%s DESC", strings.TrimPrefix(sortField, "-")))
			} else {
				db = db.Order(fmt.Sprintf("%s ASC", sortField))
			}
		}
	}

	return db
}

func GenerateQueryListMetadata(totalCount, limit, page int64) models.QueryListMetadata {
	var metadata models.QueryListMetadata

	if page <= 0 {
		page = 1
	}
	
	if limit <= 0 {
		metadata.TotalCount = totalCount
		metadata.PerPage = totalCount
		metadata.CurrentPage = 1
		metadata.TotalPage = 1
		return metadata
	}

	metadata.TotalCount = totalCount
	metadata.PerPage = limit
	metadata.CurrentPage = page
	if totalCount % limit == 0 {
		metadata.TotalPage = totalCount / limit
	} else {
		metadata.TotalPage = (totalCount / limit) + 1
	}

	return metadata
}

func GenerateFilterQueries(queriesJsonString string, sortByString string, queriesOptions models.QueriesOptions) ([]models.FilterQueries, []string, error) {
	var filters []models.FilterQueries
	var sortBy []string
	var filterJson map[string]any

	if queriesJsonString != "" {
		if err := json.Unmarshal([]byte(queriesJsonString), &filterJson); err != nil {
			return filters, sortBy, NewInvariantError(err)
		}

		for key, value := range queriesOptions.Queries {
			if v, exists := filterJson[key]; exists {
				targetColumn := key
				if value.Column != "" {
					targetColumn = value.Column
				}
				filters = append(filters, models.FilterQueries{
					Column:  targetColumn,
					Operator: value.Operators,
					Value:   v,
				})
			}
		}
	}

	if sortByString != "" {
		for sort := range strings.SplitSeq(sortByString, ",") {
			sortField := strings.TrimSpace(sort)
			if sortKey, exists := queriesOptions.Sort[strings.TrimPrefix(sortField, "-")]; exists {
				hasPrefix := strings.HasPrefix(sortField, "-")
				targetColumn := sortField
				if sortKey.Column != "" {
					targetColumn = strings.TrimSpace(sortKey.Column)
				}

				if hasPrefix {
					sortBy = append(sortBy, fmt.Sprintf("-%s", targetColumn))
				} else {
					sortBy = append(sortBy, targetColumn)
				}
			}
		}
	}

	return filters, sortBy, nil
}