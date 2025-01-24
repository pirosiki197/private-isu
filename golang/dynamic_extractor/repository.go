package dynamic_extractor

import "sync"

var queries = map[string]struct{}{}

func deleteAllQueries() {
	queries = make(map[string]struct{})
}

var querymu sync.Mutex

func addQuery(query string) {
	querymu.Lock()
	defer querymu.Unlock()
	queries[query] = struct{}{}
}

func getQueries() []string {
	ret := make([]string, 0, len(queries))
	for query := range queries {
		ret = append(ret, query)
	}
	return ret
}
