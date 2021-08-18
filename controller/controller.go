package controller

import "io"

type Controller interface {
	// komentarai komentarai
	io.Writer
	io.Closer
	Recv() ([]byte, error)
}
