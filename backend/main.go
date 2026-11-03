package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	server "orix/backend/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := loadEnvironment(); err != nil {
		logger.Error("configuration load failed", "error", err)
		os.Exit(1)
	}
	var err error
	if len(os.Args) > 1 {
		if len(os.Args) != 2 {
			logger.Error("expected one command")
			os.Exit(1)
		}
		switch os.Args[1] {
		case "retire-legacy-data":
			err = retireLegacyData()
		case "support-account-deletion":
			err = supportAccountDeletion()
		case "bootstrap-admin":
			err = bootstrapAdmin()
		case "migrate-data":
			err = migrateData()
		case "seed-test-accounts":
			err = seedTestAccounts()
		case "remove-test-accounts":
			err = removeTestAccounts()
		default:
			logger.Error("unknown command")
			os.Exit(1)
		}
	} else {
		err = run(logger)
	}
	if err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != "" {
		if err := validateLocalEmulator(); err != nil {
			return err
		}
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}
	limit := 60
	if value := os.Getenv("RATE_LIMIT_PER_MINUTE"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return errors.New("RATE_LIMIT_PER_MINUTE must be a positive integer")
		}
		limit = parsed
	}
	trustedProxies, err := server.ParseTrustedProxies(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if err != nil {
		return err
	}
	deletionWeb := server.DeletionWebConfig{
		APIKey: os.Getenv("FIREBASE_WEB_API_KEY"), AuthDomain: os.Getenv("FIREBASE_WEB_AUTH_DOMAIN"),
		ProjectID: os.Getenv("FIREBASE_PROJECT_ID"), AppID: os.Getenv("FIREBASE_WEB_APP_ID"),
		SupportEmail: os.Getenv("ACCOUNT_SUPPORT_EMAIL"), PrivacyURL: os.Getenv("PRIVACY_POLICY_URL"),
	}
	if err := deletionWeb.Validate(); err != nil {
		return err
	}
	places, err := server.NewPlaceSearch(os.Getenv("GEOCODING_BASE_URL"), os.Getenv("GEOCODING_API_TOKEN"), os.Getenv("GEOCODING_ATTRIBUTION"), os.Getenv("GEOCODING_ATTRIBUTION_URL"))
	if err != nil {
		return err
	}
	var authDependencies *server.AuthDependencies
	if os.Getenv("FIREBASE_PROJECT_ID") != "" || os.Getenv("MONGODB_URI") != "" {
		startup, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		database := os.Getenv("MONGODB_DATABASE")
		if database == "" {
			database = "orix"
		}
		dependencies, closeAuth, err := server.OpenAuth(startup, os.Getenv("FIREBASE_PROJECT_ID"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH"), os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"), os.Getenv("MONGODB_URI"), database)
		cancel()
		if err != nil {
			return err
		}
		defer closeAuth()
		authDependencies = dependencies
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app := &http.Server{
		Addr: address, Handler: server.New(server.Config{Places: places, DeletionWeb: deletionWeb, TrustedProxies: trustedProxies, AdminSecret: os.Getenv("ADMIN_API_SECRET"), Context: ctx, Auth: authDependencies, AllowedOrigins: server.ParseOrigins(os.Getenv("CORS_ORIGINS")), RequestsPerMinute: limit}, logger),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10,
	}
	failure := make(chan error, 1)
	go func() { failure <- app.ListenAndServe() }()
	logger.Info("Go migration server listening", "address", address, "version", server.Version)
	select {
	case err := <-failure:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.Shutdown(shutdown); err != nil {
			_ = app.Close()
			return err
		}
		return nil
	}
}
