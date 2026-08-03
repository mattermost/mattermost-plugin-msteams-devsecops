// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/public/shared/request"
	"github.com/mattermost/mattermost/server/v8/channels/api4"
	"github.com/mattermost/mattermost/server/v8/channels/app"
	"github.com/mattermost/mattermost/server/v8/channels/store/storetest"
	"github.com/mattermost/mattermost/server/v8/config"
	mobynetwork "github.com/moby/moby/api/types/network"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// The test containers run on a dedicated Docker network with an explicitly pinned
// subnet, chosen from the candidates below.
//
// Docker allocates its default bridge and its default address pools out of
// 172.16.0.0/12 and 10.0.0.0/8, which are also the ranges corporate VPNs most often
// advertise to their clients. When a VPN route for one of those ranges is consulted
// ahead of the host's route to the Docker bridge, containers become unreachable from
// the host even though they start correctly: the published port still accepts a
// connection because the host-side docker-proxy answers it, then no data ever comes
// back. Pinning a subnet outside those ranges keeps the suite working on such a
// machine with no change to the developer's Docker daemon or VPN.
const (
	firstTestSubnetOctet = 222
	testSubnetCount      = 10
)

// testSubnetEnvVar pins the subnet explicitly, bypassing the candidates above. Use it
// on a host where all of them are unusable.
const testSubnetEnvVar = "MSTEAMS_TEST_DOCKER_SUBNET"

// testSubnetCandidates returns the subnets to try, in order.
func testSubnetCandidates() ([]netip.Prefix, error) {
	if override := os.Getenv(testSubnetEnvVar); override != "" {
		prefix, err := netip.ParsePrefix(override)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid %s value %q, expected CIDR notation such as 192.168.222.0/24", testSubnetEnvVar, override)
		}

		return []netip.Prefix{prefix}, nil
	}

	candidates := make([]netip.Prefix, 0, testSubnetCount)
	for i := range testSubnetCount {
		addr := netip.AddrFrom4([4]byte{192, 168, byte(firstTestSubnetOctet + i), 0})
		candidates = append(candidates, netip.PrefixFrom(addr, 24))
	}

	return candidates, nil
}

// createTestNetwork creates the Docker network hosting the test containers, using the
// first candidate subnet Docker accepts.
//
// Docker rejects a network whose pool overlaps an existing one, so a single pinned
// subnet would wedge the suite whenever that range is already taken. That happens for
// two reasons worth tolerating: the developer's own network may use it, or a previous
// run may have leaked its network. A leak is possible because Ryuk is disabled (see
// setupDatabase) and an unrecovered panic in a non-main goroutine skips deferred
// cleanup. Walking the candidates keeps the suite runnable in both cases.
func createTestNetwork() (*testcontainers.DockerNetwork, error) {
	candidates, err := testSubnetCandidates()
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, subnet := range candidates {
		nw, err := tcnetwork.New(context.TODO(), tcnetwork.WithIPAM(&mobynetwork.IPAM{
			Driver: "default",
			Config: []mobynetwork.IPAMConfig{{Subnet: subnet}},
		}))
		if err == nil {
			return nw, nil
		}
		lastErr = err
	}

	return nil, errors.Wrapf(lastErr, "no usable subnet among %v, last error on %s (set %s to pin one explicitly, or run `docker network prune` to clear networks leaked by an interrupted run)", candidates, candidates[len(candidates)-1], testSubnetEnvVar)
}

// mainT is a testing.T-like structure that currently just mimics the t.Cleanup semantics.
type mainT struct {
	cleanupFunctions []func()
}

// Cleanup adds a function to be called when cleaning up.
func (mt *mainT) Cleanup(f func()) {
	mt.cleanupFunctions = append(mt.cleanupFunctions, f)
}

// Done calls all cleanup functions with defer-like semantics (last function added called first).
func (mt *mainT) Done() {
	for i := range mt.cleanupFunctions {
		f := mt.cleanupFunctions[len(mt.cleanupFunctions)-i-1]
		f()
	}
}

func (mt *mainT) Errorf(format string, args ...any) {
	fmt.Printf(format, args...)
	mt.FailNow()
}

func (mt *mainT) FailNow() {
	os.Exit(1)
}

// setupDatabase initializes a singleton Postgres testcontainer and mattermost_test database for
// use with tests.
func setupDatabase(mt *mainT) error {
	// Ryuk, the testcontainers reaper, is pinned to Docker's default bridge network
	// (reaper.go sets hc.NetworkMode = Bridge with no way to override it) and the
	// client handshakes with it from the host. It therefore cannot be moved onto the
	// network created below, and on a host where the default bridge is unreachable it
	// would hang before Postgres is ever started. Disable it and rely on the explicit
	// cleanup functions registered here, which now run even when setup panics.
	_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	nw, err := createTestNetwork()
	if err != nil {
		return err
	}
	mt.Cleanup(func() {
		if removeErr := nw.Remove(context.TODO()); removeErr != nil {
			panic(removeErr)
		}
	})

	// Setup a Postgres testcontainer for all tests.
	pgContainer, err := postgres.Run(
		context.TODO(), "docker.io/postgres:15.2-alpine",
		postgres.WithDatabase("mattermost_test"),
		postgres.WithUsername("mmuser"),
		postgres.WithPassword("mostest"),
		tcnetwork.WithNetwork([]string{"db"}, nw),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return err
	}

	containerPort, err := pgContainer.MappedPort(context.TODO(), "5432/tcp")
	if err != nil {
		return err
	}

	postgresDSN := fmt.Sprintf("postgres://mmuser:mostest@%s/mattermost_test?sslmode=disable", net.JoinHostPort("localhost", containerPort.Port()))
	_ = os.Setenv("TEST_DATABASE_POSTGRESQL_DSN", postgresDSN)

	mt.Cleanup(func() {
		if err := pgContainer.Terminate(context.TODO()); err != nil {
			panic(err)
		}
	})

	return nil
}

var server *app.Server

func getSiteURL() string {
	return fmt.Sprintf("http://localhost:%v", server.ListenAddr.Port)
}

// setupServer initializes a singleton Mattermost instance for use with tests.
func setupServer(mt *mainT) error {
	// Note that TestMain has already cleared every MM_* variable (including
	// MM_SERVICESETTINGS_SITEURL and MM_SERVICESETTINGS_LISTENADDRESS), so the
	// configuration assigned below is not overridden by the developer's shell.
	tmpDir, err := os.MkdirTemp("", "msteams")
	if err != nil {
		return err
	}
	mt.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})

	// Execute from the temporary directory to avoid polluting the developer's working
	// directory and simplify cleanup.
	err = os.Chdir(tmpDir)
	if err != nil {
		return err
	}

	// Setup a custom MM_LOCALSOCKETPATH.
	_ = os.Setenv("MM_LOCALSOCKETPATH", path.Join(tmpDir, "mattermost_local.socket"))

	// Create a test memory store and modify configuration appropriately
	configStore := config.NewTestMemoryStore()
	config := configStore.Get()
	config.PluginSettings.Directory = model.NewPointer(path.Join(tmpDir, "plugins"))
	config.PluginSettings.ClientDirectory = model.NewPointer(path.Join(tmpDir, "client"))
	config.ServiceSettings.ListenAddress = model.NewPointer("localhost:0")
	config.TeamSettings.MaxUsersPerTeam = model.NewPointer(10000)
	config.LocalizationSettings.SetDefaults()
	config.SqlSettings = *storetest.MakeSqlSettings("postgres")
	config.ServiceSettings.SiteURL = model.NewPointer("http://example.com/")
	config.LogSettings.EnableConsole = model.NewPointer(true)
	config.LogSettings.EnableFile = model.NewPointer(false)
	config.LogSettings.ConsoleLevel = model.NewPointer("DEBUG")
	config.ServiceSettings.EnableLocalMode = model.NewPointer(true)
	config.ServiceSettings.LocalModeSocketLocation = model.NewPointer(path.Join(tmpDir, "mattermost_local.socket"))
	config.ServiceSettings.EnableDeveloper = model.NewPointer(true)
	config.ServiceSettings.EnableTesting = model.NewPointer(true)
	config.FileSettings.Directory = model.NewPointer(path.Join(tmpDir, "data"))

	_, _, err = configStore.Set(config)
	if err != nil {
		return err
	}

	// Create a logger to override
	testLogger, err := mlog.NewLogger()
	if err != nil {
		return err
	}
	testLogger.LockConfiguration()

	// Initialize the server with app and api4 interfaces.
	options := []app.Option{
		app.ConfigStore(configStore),
	}

	server, err = app.NewServer(options...)
	if err != nil {
		return err
	}

	_, err = api4.Init(server)
	if err != nil {
		return err
	}

	err = server.Start()
	if err != nil {
		return err
	}
	mt.Cleanup(func() {
		server.Shutdown()
	})

	ap := app.New(app.ServerConnector(server.Channels()))

	// Setup the first user immediately.
	username := model.NewUsername()
	user := &model.User{
		Email:         fmt.Sprintf("%s@example.com", username),
		Username:      username,
		Password:      "password",
		EmailVerified: true,
	}

	_, appErr := ap.CreateUser(request.EmptyContext(testLogger), user)
	if appErr != nil {
		return appErr
	}

	return nil
}

var setupReattachEnvironmentOnce sync.Once

// setupReattachEnvironment is used by the test helper to initialize the infrastructure for running
// reattached plugin tests exactly once (per package).
//
// Note that while we assert on the given *testing.T, we setup cleanup functions on the global
// *mainT to clean up once at termination.
func setupReattachEnvironment(mt *mainT) {
	setupReattachEnvironmentOnce.Do(func() {
		err := setupDatabase(mt)
		require.NoError(mt, err)

		err = setupServer(mt)
		require.NoError(mt, err)
	})
}

// TestMain is run before any tests within this package and helps setup a mainT for global cleanup
// if needed.
func TestMain(m *testing.M) {
	// Clear any MM_* environment variables inherited from the developer's shell.
	// The test configuration is a memory store that applies environment overrides
	// on top of the settings assigned in setupServer, so variables such as
	// MM_SERVICESETTINGS_SITEURL or MM_SQLSETTINGS_DATASOURCE would otherwise change
	// how the test server is configured. This is hardening rather than the fix for
	// the false green below: it makes local runs hermetic and match CI, which runs
	// with a clean environment.
	clearMattermostEnv()

	// This is the MM-69712 fix. Run the suite inside a helper so deferred cleanup
	// executes before os.Exit, and so a panic during setup or cleanup propagates and
	// fails the run. The previous "defer os.Exit(status)" pattern swallowed such
	// panics: os.Exit running during panic unwinding terminated the process with
	// status 0, so a setup failure (for example the postgres testcontainer being
	// unreachable) produced no test output and a green run.
	os.Exit(func() int {
		mt := new(mainT)
		defer mt.Done()

		setupReattachEnvironment(mt)

		// This actually runs the tests.
		return m.Run()
	}())
}

// clearMattermostEnv unsets every MM_* environment variable so a developer's
// local shell configuration cannot override the hermetic test configuration.
func clearMattermostEnv() {
	for _, kv := range os.Environ() {
		if key, _, ok := strings.Cut(kv, "="); ok && strings.HasPrefix(key, "MM_") {
			_ = os.Unsetenv(key)
		}
	}
}
