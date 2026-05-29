package services

import "os"

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func openAppend(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}
