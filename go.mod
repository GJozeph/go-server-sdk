module github.com/launchdarkly/go-server-sdk/v7

go 1.18

require (
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gregjones/httpcache v0.0.0-20171119193500-2bcd89a1743f
	github.com/launchdarkly/ccache v1.1.0
	github.com/launchdarkly/eventsource v1.6.2
	github.com/launchdarkly/go-jsonstream/v3 v3.0.0
	github.com/launchdarkly/go-ntlm-proxy-auth v1.0.1
	github.com/launchdarkly/go-sdk-common/v3 v3.1.0
	github.com/launchdarkly/go-sdk-events/v3 v3.2.0
	github.com/launchdarkly/go-server-sdk-evaluation/v3 v3.0.0
	github.com/launchdarkly/go-test-helpers/v3 v3.0.2
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/trace v1.22.0
	golang.org/x/exp v0.0.0-20220823124025-807a23277127
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/ghodss/yaml.v1 v1.0.0
)

// TEMPORARY CHANGES TO TEST OTEL
//require (
//	go.opentelemetry.io/otel v1.22.0
//	go.opentelemetry.io/otel/trace v1.22.0
//)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/launchdarkly/go-ntlmssp v1.0.1 // indirect
	github.com/launchdarkly/go-semver v1.0.2 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	go.opentelemetry.io/otel/metric v1.22.0 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
