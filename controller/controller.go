package controller

type Controller interface {
	Start(rx, tx chan []byte)
	Stop()
}
