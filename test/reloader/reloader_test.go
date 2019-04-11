package reloadertest

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	natsreloader "github.com/nats-io/nats-operator/pkg/reloader"
)

var configContents = `port = 2222`

func TestReloader(t *testing.T) {
	// Setup a pidfile that points to us
	pid := os.Getpid()
	pidfile, err := ioutil.TempFile(os.TempDir(), "nats-pid-")
	if err != nil {
		t.Fatal(err)
	}

	p := fmt.Sprintf("%d", pid)
	if _, err := pidfile.WriteString(p); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pidfile.Name())

	// Create tempfile with contents, then update it
	nconfig := &natsreloader.Config{
		PidFile:     pidfile.Name(),
		ConfigFiles: []string{},
	}

	var configFiles []*os.File
	for i := 0; i < 2; i++ {
		configFile, err := ioutil.TempFile(os.TempDir(), "nats-conf-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(configFile.Name())

		if _, err := configFile.WriteString(configContents); err != nil {
			t.Fatal(err)
		}
		configFiles = append(configFiles, configFile)
		nconfig.ConfigFiles = append(nconfig.ConfigFiles, configFile.Name())
	}

	r, err := natsreloader.NewReloader(nconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	var signals = 0

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Signal handling.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)

		// Success when receiving the first signal
		for range c {
			signals++
		}
	}()

	go func() {
		for _, configfile := range configFiles {
			for i := 0; i < 5; i++ {
				// Append some more stuff to the config
				if _, err := configfile.WriteAt([]byte(configContents), 0); err != nil {
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
		cancel()
	}()

	err = r.Run(ctx)
	if err != nil && err != context.Canceled {
		t.Fatal(err)
	}
	// We should have gotten only one signal for each configuration file
	if signals == len(configFiles) {
		t.Fatalf("Timed out waiting for reloading signal")
	}
}
