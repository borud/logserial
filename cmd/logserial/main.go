// Package main is the main package
package main

import (
	"bufio"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/borud/logserial/pkg/model"
	"github.com/borud/logserial/pkg/store"
	"github.com/borud/logserial/pkg/store/sqlitestore"
	"go.bug.st/serial"
)

const dbFile = "logserial.db"

func main() {
	if len(os.Args) < 2 {
		log.Fatal("please provide list of serial ports")
	}

	db, err := sqlitestore.New(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, devicePath := range os.Args[1:] {
		go logSerial(db, devicePath)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}

func logSerial(db store.Store, devicePath string) {
	for {
		port, err := serial.Open(devicePath, &serial.Mode{
			BaudRate: 115200,
			DataBits: 8,
			Parity:   serial.NoParity,
			StopBits: serial.OneStopBit,
		})
		if err != nil {
			// if we fail to connect we just keep retrying ever 100ms
			slog.Debug("unable to open serial port", "device", devicePath, "err", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		var msg string
		scanner := bufio.NewScanner(port)
		for scanner.Scan() {
			msg = scanner.Text()

			db.Log(model.Message{
				TS:     uint64(time.Now().UnixMilli()),
				Device: devicePath,
				Msg:    msg,
			})

			slog.Info("", "device", devicePath, "msg", scanner.Text())
		}

		slog.Error("lost connection", "device", devicePath)
	}
}
