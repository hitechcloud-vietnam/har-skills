package main

import (
	"fmt"
	"os"

	"github.com/hitechcloud-vietnam/har-skills"
)

func main() {

	harFilePath := "./data/www.google.com.har"

	// Usage 1: Parse HAR file bytes.
	harFileBytes, err := os.ReadFile(harFilePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	harFile, err := har.ParseHar(harFileBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(harFile)

	// Usage 2: Parse a HAR file from the specified path.
	harFile002, err := har.ParseHarFile(harFilePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(harFile002)

}
