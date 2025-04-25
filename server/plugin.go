package main

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/mattermost/mattermost-plugin-msteams-devsecops/server/msteams"
	"github.com/mattermost/mattermost-plugin-msteams-devsecops/server/store/pluginstore"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/sirupsen/logrus"
)

const (
	pluginID                = "com.mattermost.plugin-msteams-devsecops"
	checkCredentialsJobName = "check_credentials" //#nosec G101 -- This is a false positive
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// client is the Mattermost server API client.
	client *pluginapi.Client

	// msteamsAppClient is the client used to communicate with the Microsoft Teams API.
	msteamsAppClient      msteams.Client
	msteamsAppClientMutex sync.RWMutex

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// handlers for incoming Rest API requests
	apiHandler *API

	// plugin KV store
	pluginStore *pluginstore.PluginStore

	// tabAppJWTKeyFunc is the keyfunc.Keyfunc used to verify JWTs from Microsoft Teams
	tabAppJWTKeyFunc  keyfunc.Keyfunc
	cancelKeyFunc     context.CancelFunc
	cancelKeyFuncLock sync.Mutex

	// checkCredentialsJob is a job that periodically checks credentials and permissions against the MS Graph API
	checkCredentialsJob *cluster.Job

	// clientReconnectCtx and clientReconnectCancel are used to control the client reconnection goroutine
	clientReconnectCtx    context.Context
	clientReconnectCancel context.CancelFunc
	clientReconnectLock   sync.Mutex
}

func (p *Plugin) GetClientForApp() msteams.Client {
	p.msteamsAppClientMutex.RLock()
	defer p.msteamsAppClientMutex.RUnlock()

	return p.msteamsAppClient
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	logger := logrus.StandardLogger()
	pluginapi.ConfigureLogrus(logger, p.client)

	config := p.client.Configuration.GetConfig()
	license := p.client.System.GetLicense()
	if !pluginapi.IsE20LicensedOrDevelopment(config, license) {
		return errors.New("this plugin requires an enterprise license")
	}

	p.apiHandler = NewAPI(p)

	p.pluginStore = pluginstore.NewPluginStore(p.API)

	go p.start(false)

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	p.stop(false)
	return nil
}

func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.apiHandler.ServeHTTP(w, r)
}

func (p *Plugin) start(isRestart bool) {
	// set up JWK for verifying JWTs from Microsoft Teams
	p.cancelKeyFuncLock.Lock()
	if !isRestart && p.cancelKeyFunc == nil {
		p.tabAppJWTKeyFunc, p.cancelKeyFunc = setupJWKSet()
	}
	p.cancelKeyFuncLock.Unlock()

	// Initialize context for client reconnection
	p.clientReconnectLock.Lock()
	if p.clientReconnectCtx == nil {
		p.clientReconnectCtx, p.clientReconnectCancel = context.WithCancel(context.Background())
	}
	p.clientReconnectLock.Unlock()

	// connect to the Microsoft Teams API
	err := p.connectTeamsAppClient()
	if err != nil {
		return
	}

	if !p.getConfiguration().DisableCheckCredentials {
		checkCredentialsJob, jobErr := cluster.Schedule(
			p.API,
			checkCredentialsJobName,
			cluster.MakeWaitForRoundedInterval(24*time.Hour),
			p.checkCredentials,
		)
		if jobErr != nil {
			p.API.LogError("error in scheduling the check credentials job", "error", jobErr)
			return
		}
		p.checkCredentialsJob = checkCredentialsJob

		// Run the job above right away
		go p.checkCredentials()
	}

	p.API.LogDebug("plugin started")
}

func (p *Plugin) stop(isRestart bool) {
	if p.checkCredentialsJob != nil {
		if err := p.checkCredentialsJob.Close(); err != nil {
			p.API.LogError("Failed to close background check credentials job", "error", err)
		}
		p.checkCredentialsJob = nil
	}

	// Cancel the client reconnection context if not restarting
	if !isRestart {
		p.clientReconnectLock.Lock()
		if p.clientReconnectCancel != nil {
			p.clientReconnectCancel()
			p.clientReconnectCtx = nil
			p.clientReconnectCancel = nil
		}
		p.clientReconnectLock.Unlock()

		p.cancelKeyFuncLock.Lock()
		if p.cancelKeyFunc != nil {
			p.cancelKeyFunc()
			p.cancelKeyFunc = nil
		}
		p.cancelKeyFuncLock.Unlock()
	}
}

func (p *Plugin) restart() {
	p.stop(true)
	p.start(true)
}

func (p *Plugin) connectTeamsAppClient() error {
	p.msteamsAppClientMutex.Lock()
	defer p.msteamsAppClientMutex.Unlock()

	// We don't currently support reconnecting with a new configuration: a plugin restart is
	// required.
	if p.msteamsAppClient != nil {
		return nil
	}

	p.msteamsAppClient = msteams.NewApp(
		p.getConfiguration().TenantID,
		p.getConfiguration().AppClientID,
		p.getConfiguration().AppClientSecret,
		&p.client.Log,
	)

	err := p.msteamsAppClient.Connect()
	if err != nil {
		p.API.LogError("Unable to connect to the app client", "error", err)
		return err
	}

	// Get a local copy of the context to use in the goroutine
	p.clientReconnectLock.Lock()
	ctx := p.clientReconnectCtx
	p.clientReconnectLock.Unlock()

	// Start a goroutine to periodically reconnect the client to refresh the token
	go func(ctx context.Context) {
		p.API.LogDebug("Starting client reconnection goroutine")

		reconnectInterval := 12 * time.Hour
		ticker := time.NewTicker(reconnectInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				p.API.LogDebug("Client reconnection goroutine stopped")
				return
			case <-ticker.C:
				p.API.LogDebug("Reconnecting MS Teams app client to refresh token")
				p.msteamsAppClientMutex.Lock()
				if p.msteamsAppClient != nil {
					if err := p.msteamsAppClient.Connect(); err != nil {
						p.API.LogError("Failed to reconnect MS Teams app client", "error", err)
					} else {
						p.API.LogDebug("Successfully reconnected MS Teams app client")
					}
				}
				p.msteamsAppClientMutex.Unlock()
			}
		}
	}(ctx)

	return nil
}
