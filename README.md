# govkdoc

Go impl of vk.com api workflow to share uploaded documents

Example (auth token required with "docs,wall" permissions)

package main

import (
	"govkdoc"
)

func main() {
	vc := govkdoc.NewVkConn("<insert valid vk.com auth token>")
	files := []string{
		"upload_img.gif",
	}
	for _, fileName := range files {
		vc.WallShareFileAsDoc("file", fileName)
		//<-time.After(30 * time.Second)
	}
}
