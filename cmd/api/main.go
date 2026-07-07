package main

import (
	"CrudTutorialProject/internal/app"
	"fmt"
	"os"
)

func main() {
	if err := app.Run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "application faild %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}
