package models

type QueryListMetadata struct {
	TotalCount 	int64 
	TotalPage  	int64
	CurrentPage int64 
	PerPage    	int64
	NextCursor 	*string
	PrevCursor 	*string
}

type Queries struct {
	Limit  int64
	Page   int64
	Sorters []string
	Filters []FilterQueries
}

type FilterQueries struct {
	Column  string
	Operator string
	Value   any
}

type QueriesItem struct {
	Column  string
	Operators string
}

type SortItem struct {
	Column	string
}

type QueriesOptions struct {
	Queries 	map[string]QueriesItem
	Sort		map[string]SortItem
}