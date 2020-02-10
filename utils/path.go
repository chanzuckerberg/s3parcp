package utils

// AddTrailingSlash adds a / to the end of a string if there isn't one there
func AddTrailingSlash(path string) string {
	if path[len(path)-1] != '/' {
		return path + "/"
	}
	return path
}
