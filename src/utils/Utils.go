package utils

import "os"

func GetFiles() ([]string, error) {
	dir, err := os.Open(".")
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func ReadFile(filename string) []byte {
	content, err := os.ReadFile(filename)

	if err != nil {
		// fmt.Println("could not load file:", err)
		// os.Exit(1)
	}
	return content
}
