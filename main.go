// The purpose of this example is to show how to integrate with zap.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	adapter "github.com/axiomhq/axiom-go/adapters/zap"
	"github.com/pkg/errors"
)

func main() {
	// Export "AXIOM_DATASET" in addition to the required environment variables.

	// Prepare WriteSyncer
	ws := zapcore.Lock(WrappedWriteSyncer{os.Stdout})

	// Create the core
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	stdOutCore := zapcore.NewCore(enc, ws, zapcore.DebugLevel)

	axiomCore, err := adapter.New()
	if err != nil {
		log.Fatal(err)
	}

	core := zapcore.NewTee(
		stdOutCore,
		axiomCore,
	)
	logger := zap.New(core)

	// 3. Have all logs flushed before the application exits.
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			log.Fatal(syncErr)
		}
	}()

	logger = logger.With(zap.String("application", "logger"), zap.String("network", "bjartek"))

	err = fmt.Errorf("Foobar")

	newErr := errors.Wrapf(err, "oh boy %s", "baaaz")

	// 4. Log ⚡
	logger.Info("This is awesome!", zap.String("mood", "hyped"))
	logger.Warn("This is no that awesome...", zap.String("mood", "worried"))
	logger.Error("This is rather bad.", zap.String("mood", "depressed"), zap.Error(newErr))

	run(func(i int) (*time.Duration, error) {
		logger.Info("This is awesome!", zap.String("mood", "hyped"))
		logger.Warn("This is no that awesome...", zap.String("mood", "worried"))
		logger.Error("This is rather bad.", zap.String("mood", "depressed"), zap.Error(newErr))
		return nil, nil
	}, time.Second)

}

// WrappedWriteSyncer is a helper struct implementing zapcore.WriteSyncer to
// wrap a standard os.Stdout handle, giving control over the WriteSyncer's
// Sync() function. Sync() results in an error on Windows in combination with
// os.Stdout ("sync /dev/stdout: The handle is invalid."). WrappedWriteSyncer
// simply does nothing when Sync() is called by Zap.
type WrappedWriteSyncer struct {
	file *os.File
}

func (mws WrappedWriteSyncer) Write(p []byte) (n int, err error) {
	return mws.file.Write(p)
}
func (mws WrappedWriteSyncer) Sync() error {
	return nil
}

type ScheduleFunction func(iteration int) (*time.Duration, error)

/// A function to run a provided callback every time.Duration until stopped.
///
/// The callback can return a new sleep duration if it wants to or an error
/// If error is returned we panic
///
/// If a signal is sent to the process then we will exit with success (0) once the current iteration is over.
func run(fn ScheduleFunction, sleep time.Duration) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSEGV)
	defer func() {
		signal.Stop(c)
	}()

	for i := 1; ; i++ {
		select {
		case <-c:
			os.Exit(0)
		default:
			fnSleep, err := fn(i)
			if err != nil {
				panic(err)
			}

			if fnSleep != nil {
				time.Sleep(*fnSleep)
			} else {
				time.Sleep(sleep)
			}
		}
	}
}
