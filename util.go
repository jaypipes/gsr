package gsr

func contains(search string, in []*Endpoint) bool {
	for _, s := range in {
		if s.Address == search {
			return true
		}
	}
	return false
}

func containsAll(all []string, in []*Endpoint) bool {
	for _, each := range all {
		if !contains(each, in) {
			return false
		}
	}
	return true
}
