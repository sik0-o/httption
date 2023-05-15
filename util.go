package httption

func mergeMaps(source map[string]string, maps ...map[string]string) {
	for _, mm := range maps {
		for k, v := range mm {
			source[k] = v
		}
	}
}

func mergedMaps(maps ...map[string]string) map[string]string {
	merged := make(map[string]string)
	mergeMaps(merged, maps...)

	return merged
}
