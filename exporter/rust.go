package exporter

import (
	"github.com/gorcon/websocket"
	log "github.com/sirupsen/logrus"
)

func (e *Exporter) connectToRust() (*websocket.Conn, error) {
	return websocket.Dial(e.rustAddr, e.options.Password)
}

func (e *Exporter) doRustCmd(cmd string) (string, error) {
	log.Debugf("c.Execute() - running command: %s", cmd)
	res, err := e.conn.Execute(cmd)
	if err != nil {
		log.Debugf("c.Execute() - res: %s, err: %s", res, err)
	}
	log.Debugf("c.Execute() - done")
	return res, err
}
