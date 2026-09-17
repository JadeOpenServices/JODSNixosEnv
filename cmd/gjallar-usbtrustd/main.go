package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
	"github.com/bakanura/gjallarOS/internal/usbtrust/broker"
	"github.com/bakanura/gjallarOS/internal/usbtrust/daemon"
	"github.com/bakanura/gjallarOS/internal/usbtrust/readmodel"
	"github.com/bakanura/gjallarOS/internal/usbtrust/usbguardsource"
)

type config struct {
	socketPath     string
	ownerUID       uint32
	oddcRoot       string
	oddcModel      string
	stateDir       string
	tpmHandle      string
	usbguardBinary string
	requestTimeout time.Duration
}

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	os.Exit(
		run(
			ctx,
			os.Args[1:],
			os.Stdout,
			os.Stderr,
		),
	)
}

func run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	cfg, err := parseConfig(args, stderr)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"FAIL: %v\n",
			err,
		)
		return 2
	}

	reader, err := buildReader(cfg)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"FAIL: configure USB trust read model: %v\n",
			err,
		)
		return 1
	}

	server := daemon.Server{
		SocketPath:     cfg.socketPath,
		RequestTimeout: cfg.requestTimeout,
		Handler: broker.Handler{
			OwnerUID: cfg.ownerUID,
			Reader:   reader,
		},
		OnError: func(err error) {
			fmt.Fprintf(
				stderr,
				"WARN: %v\n",
				err,
			)
		},
	}

	fmt.Fprintf(
		stdout,
		"PASS: gjallar-usbtrustd read-only broker listening on %s\n",
		cfg.socketPath,
	)

	if err := server.Serve(ctx); err != nil {
		fmt.Fprintf(
			stderr,
			"FAIL: USB trust daemon: %v\n",
			err,
		)
		return 1
	}

	return 0
}

func parseConfig(
	args []string,
	stderr io.Writer,
) (config, error) {
	var cfg config
	var ownerUIDText string

	flags := flag.NewFlagSet(
		"gjallar-usbtrustd",
		flag.ContinueOnError,
	)
	flags.SetOutput(stderr)

	flags.StringVar(
		&cfg.socketPath,
		"socket",
		"",
		"absolute Unix control socket path",
	)

	flags.StringVar(
		&ownerUIDText,
		"owner-uid",
		"",
		"desktop user UID allowed read-only access",
	)

	flags.StringVar(
		&cfg.oddcRoot,
		"oddc-root",
		"",
		"absolute ODDC catalog root",
	)

	flags.StringVar(
		&cfg.oddcModel,
		"oddc-model",
		"",
		"already-selected ODDC model ID",
	)

	flags.StringVar(
		&cfg.stateDir,
		"state-dir",
		"",
		"absolute signed USB trust state directory",
	)

	flags.StringVar(
		&cfg.tpmHandle,
		"tpm-handle",
		"",
		"TPM persistent signing-key handle",
	)

	flags.StringVar(
		&cfg.usbguardBinary,
		"usbguard-binary",
		"usbguard",
		"USBGuard CLI binary",
	)

	flags.DurationVar(
		&cfg.requestTimeout,
		"request-timeout",
		daemon.DefaultRequestTimeout,
		"maximum time per broker connection",
	)

	if err := flags.Parse(args); err != nil {
		return config{}, err
	}

	if flags.NArg() != 0 {
		return config{}, fmt.Errorf(
			"unexpected positional arguments: %s",
			strings.Join(flags.Args(), " "),
		)
	}

	if strings.TrimSpace(cfg.socketPath) == "" {
		return config{}, fmt.Errorf(
			"--socket is required",
		)
	}

	if !filepath.IsAbs(cfg.socketPath) {
		return config{}, fmt.Errorf(
			"--socket must be absolute",
		)
	}

	if strings.TrimSpace(ownerUIDText) == "" {
		return config{}, fmt.Errorf(
			"--owner-uid is required",
		)
	}

	ownerUID, err := strconv.ParseUint(
		ownerUIDText,
		10,
		32,
	)
	if err != nil ||
		ownerUID > math.MaxUint32 {
		return config{}, fmt.Errorf(
			"invalid --owner-uid %q",
			ownerUIDText,
		)
	}

	cfg.ownerUID = uint32(ownerUID)

	if strings.TrimSpace(cfg.oddcRoot) == "" {
		return config{}, fmt.Errorf(
			"--oddc-root is required",
		)
	}

	if !filepath.IsAbs(cfg.oddcRoot) {
		return config{}, fmt.Errorf(
			"--oddc-root must be absolute",
		)
	}

	if strings.TrimSpace(cfg.oddcModel) == "" {
		return config{}, fmt.Errorf(
			"--oddc-model is required",
		)
	}

	if strings.TrimSpace(cfg.stateDir) == "" {
		return config{}, fmt.Errorf(
			"--state-dir is required",
		)
	}

	if !filepath.IsAbs(cfg.stateDir) {
		return config{}, fmt.Errorf(
			"--state-dir must be absolute",
		)
	}

	if strings.TrimSpace(cfg.usbguardBinary) == "" {
		return config{}, fmt.Errorf(
			"--usbguard-binary may not be empty",
		)
	}

	if cfg.requestTimeout <= 0 {
		return config{}, fmt.Errorf(
			"--request-timeout must be positive",
		)
	}

	return cfg, nil
}

func buildReader(
	cfg config,
) (readmodel.Reader, error) {
	var signer usbtrust.Signer

	if strings.TrimSpace(cfg.tpmHandle) != "" {
		tpmSigner, err := usbtrust.NewTPMSigner(
			cfg.tpmHandle,
		)
		if err != nil {
			return readmodel.Reader{}, err
		}

		signer = tpmSigner
	}

	return readmodel.Reader{
		Resolved: readmodel.NewCatalogResolvedSource(
			cfg.oddcRoot,
			cfg.oddcModel,
		),
		Observations: usbguardsource.LiveSource{
			Runner: usbguardsource.ExecRunner{},
			Binary: cfg.usbguardBinary,
		},
		Trust: readmodel.NewVerifiedTrustSource(
			cfg.stateDir,
			signer,
		),
	}, nil
}
