// Package ldfiledata allows the LaunchDarkly client to read feature flag data from a file.
package ldfiledata

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"gopkg.in/ghodss/yaml.v1"

	ld "gopkg.in/launchdarkly/go-server-sdk.v4"
)

// FileDataSourceOption is the interface for optional configuration parameters that can be
// passed to NewFileDataSourceFactory. These include FilePaths and UseLogger.
type FileDataSourceOption interface {
	apply(fp *fileDataSource) error
}

type filePathsOption struct {
	paths []string
}

func (o filePathsOption) apply(fs *fileDataSource) error {
	abs, err := absFilePaths(o.paths)
	if err != nil {
		return err
	}
	fs.absFilePaths = append(fs.absFilePaths, abs...)
	return nil
}

// FilePaths creates an option for to NewFileDataSourceFactory, to specify the input
// data files. The paths may be any number of absolute or relative file paths.
func FilePaths(paths ...string) FileDataSourceOption {
	return filePathsOption{paths}
}

type loggerOption struct {
	logger ld.Logger
}

func (o loggerOption) apply(fs *fileDataSource) error {
	fs.logger = o.logger
	return nil
}

// UseLogger creates an option for NewFileDataSourceFactory, to specify where to send
// log output. If not specified, a log.Logger is used.
func UseLogger(logger ld.Logger) FileDataSourceOption {
	return loggerOption{logger}
}

// ReloaderFactory is a function type used with UseReloader, to specify a mechanism for detecting when
// data files should be reloaded. Its standard implementation is in the ldfilewatch package.
type ReloaderFactory func(paths []string, logger ld.Logger, reload func(), closeCh <-chan struct{}) error

type reloaderOption struct {
	reloaderFactory ReloaderFactory
}

func (o reloaderOption) apply(fs *fileDataSource) error {
	fs.reloaderFactory = o.reloaderFactory
	return nil
}

// UseReloader creates an option for NewFileDataSourceFactory, to specify a mechanism for reloading
// data files. It is normally used with the ldfilewatch package, as follows:
//
//     ldfiledata.UseReloader(ldfilewatch.WatchFiles)
func UseReloader(reloaderFactory ReloaderFactory) FileDataSourceOption {
	return reloaderOption{reloaderFactory}
}

type fileDataSource struct {
	store           ld.FeatureStore
	reloaderFactory ReloaderFactory
	logger          ld.Logger
	isInitialized   bool
	absFilePaths    []string
	readyCh         chan<- struct{}
	readyOnce       sync.Once
	closeOnce       sync.Once
	closeReloaderCh chan struct{}
}

// NewFileDataSourceFactory returns a function that allows the LaunchDarkly client to read feature
// flag data from a file or files. You must store this function in the UpdateProcessorFactory
// property of your client configuration before creating the client:
//
//     fileSource, err := ldfiledata.NewFileDataSourceFactory(
//         ldfiledata.FilePaths("./test-data/my-flags.json"))
//     ldConfig := ld.DefaultConfig
//     ldConfig.UpdateProcessorFactory = fileSource
//     ldClient := ld.MakeCustomClient(mySdkKey, ldConfig, 5*time.Second)
//
// Use FilePaths to specify any number of file paths. The files are not actually loaded until the
// client starts up. At that point, if any file does not exist or cannot be parsed, the FileDataSource
// will log an error and will not load any data.
//
// Files may contain either JSON or YAML; if the first non-whitespace character is '{', the file is parsed
// as JSON, otherwise it is parsed as YAML. The file data should consist of an object with up to three
// properties:
//
// - "flags": Feature flag definitions.
//
// - "flagValues": Simplified feature flags that contain only a value.
//
// - "segments": User segment definitions.
//
// The format of the data in "flags" and "segments" is defined by the LaunchDarkly application and is
// subject to change. Rather than trying to construct these objects yourself, it is simpler to request
// existing flags directly from the LaunchDarkly server in JSON format, and use this output as the starting
// point for your file. In Linux you would do this:
//
//     curl -H "Authorization: <your sdk key>" https://app.launchdarkly.com/sdk/latest-all
//
// The output will look something like this (but with many more properties):
//
//     {
//       "flags": {
//         "flag-key-1": {
//           "key": "flag-key-1",
//           "on": true,
//           "variations": [ "a", "b" ]
//         }
//       },
//       "segments": {
//         "segment-key-1": {
//           "key": "segment-key-1",
//           "includes": [ "user-key-1" ]
//         }
//       }
//     }
//
// Data in this format allows the SDK to exactly duplicate all the kinds of flag behavior supported by
// LaunchDarkly. However, in many cases you will not need this complexity, but will just want to set
// specific flag keys to specific values. For that, you can use a much simpler format:
//
//     {
//       "flagValues": {
//         "my-string-flag-key": "value-1",
//         "my-boolean-flag-key": true,
//         "my-integer-flag-key": 3
//       }
//     }
//
// Or, in YAML:
//
//     flagValues:
//       my-string-flag-key: "value-1"
//       my-boolean-flag-key: true
//       my-integer-flag-key: 3
//
// It is also possible to specify both "flags" and "flagValues", if you want some flags to have simple
// values and others to have complex behavior. However, it is an error to use the same flag key or
// segment key more than once, either in a single file or across multiple files.
//
// If the data source encounters any error in any file-- malformed content, a missing file, or a
// duplicate key-- it will not load flags from any of the files.
func NewFileDataSourceFactory(options ...FileDataSourceOption) ld.UpdateProcessorFactory {
	return func(sdkKey string, config ld.Config) (ld.UpdateProcessor, error) {
		return newFileDataSource(config.FeatureStore, options...)
	}
}

func newFileDataSource(featureStore ld.FeatureStore, options ...FileDataSourceOption) (*fileDataSource, error) {
	if featureStore == nil {
		return nil, fmt.Errorf("featureStore must not be nil")
	}
	fs := &fileDataSource{
		store: featureStore,
	}
	for _, o := range options {
		err := o.apply(fs)
		if err != nil {
			return nil, err
		}
	}
	if fs.logger == nil {
		fs.logger = log.New(os.Stderr, "[LaunchDarkly FileDataSource] ", log.LstdFlags)
	}
	return fs, nil
}

// Initialized is used internally by the LaunchDarkly client.
func (fs *fileDataSource) Initialized() bool {
	return fs.isInitialized
}

// Start is used internally by the LaunchDarkly client.
func (fs *fileDataSource) Start(closeWhenReady chan<- struct{}) {
	fs.readyCh = closeWhenReady
	fs.reload()

	// If there is no reloader, then we signal readiness immediately regardless of whether the
	// data load succeeded or failed.
	if fs.reloaderFactory == nil {
		fs.signalStartComplete(fs.isInitialized)
		return
	}

	// If there is a reloader, and if we haven't yet successfully loaded data, then the
	// readiness signal will happen the first time we do get valid data (in reload).
	fs.closeReloaderCh = make(chan struct{})
	err := fs.reloaderFactory(fs.absFilePaths, fs.logger, fs.reload, fs.closeReloaderCh)
	if err != nil {
		fs.logger.Printf("ERROR: Unable to start reloader: %s\n", err)
	}
}

// Reload tells the data source to immediately attempt to reread all of the configured source files
// and update the feature flag state. If any file cannot be loaded or parsed, the flag state will not
// be modified.
func (fs *fileDataSource) reload() {
	filesData := make([]fileData, 0)
	for _, path := range fs.absFilePaths {
		data, err := readFile(path)
		if err == nil {
			filesData = append(filesData, data)
		} else {
			fs.logger.Printf("ERROR: Unable to load flags: %s [%s]", err, path)
			return
		}
	}
	storeData, err := mergeFileData(filesData...)
	if err == nil {
		err = fs.store.Init(storeData)
		fs.signalStartComplete(true)
	}
	if err != nil {
		fs.logger.Printf("ERROR: %s", err)
	}
}

func (fs *fileDataSource) signalStartComplete(succeeded bool) {
	fs.readyOnce.Do(func() {
		fs.isInitialized = succeeded
		if fs.readyCh != nil {
			close(fs.readyCh)
		}
	})
}

func absFilePaths(paths []string) ([]string, error) {
	absPaths := make([]string, 0)
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("unable to determine absolute path for '%s'", p)
		}
		absPaths = append(absPaths, absPath)
	}
	return absPaths, nil
}

type fileData struct {
	Flags      *map[string]ld.FeatureFlag //nolint:megacheck
	FlagValues *map[string]interface{}
	Segments   *map[string]ld.Segment //nolint:megacheck
}

func insertData(all map[ld.VersionedDataKind]map[string]ld.VersionedData, kind ld.VersionedDataKind, key string,
	data ld.VersionedData) error {
	if _, exists := all[kind][key]; exists {
		return fmt.Errorf("%s '%s' is specified by multiple files", kind.GetNamespace(), key)
	}
	all[kind][key] = data
	return nil
}

func readFile(path string) (fileData, error) {
	var data fileData
	var rawData []byte
	var err error
	if rawData, err = ioutil.ReadFile(path); err != nil { // nolint:gosec // G304: ok to read file into variable
		return data, fmt.Errorf("unable to read file: %s", err)
	}
	if detectJSON(rawData) {
		err = json.Unmarshal(rawData, &data)
	} else {
		err = yaml.Unmarshal(rawData, &data)
	}
	if err != nil {
		err = fmt.Errorf("error parsing file: %s", err)
	}
	return data, err
}

func detectJSON(rawData []byte) bool {
	// A valid JSON file for our purposes must be an object, i.e. it must start with '{'
	return strings.HasPrefix("{", strings.TrimLeftFunc(string(rawData), unicode.IsSpace))
}

func mergeFileData(allFileData ...fileData) (map[ld.VersionedDataKind]map[string]ld.VersionedData, error) {
	all := map[ld.VersionedDataKind]map[string]ld.VersionedData{
		ld.Features: {},
		ld.Segments: {},
	}
	for _, d := range allFileData {
		if d.Flags != nil {
			for key, f := range *d.Flags {
				data := f
				if err := insertData(all, ld.Features, key, &data); err != nil {
					return nil, err
				}
			}
		}
		if d.FlagValues != nil {
			for key, f := range *d.FlagValues {
				zeroVariation := 0
				data := ld.FeatureFlag{ //nolint:megacheck
					Key:         key,
					Variations:  []interface{}{f},
					On:          true,
					Fallthrough: ld.VariationOrRollout{Variation: &zeroVariation}, //nolint:megacheck
				}
				if err := insertData(all, ld.Features, key, &data); err != nil {
					return nil, err
				}
			}
		}
		if d.Segments != nil {
			for key, s := range *d.Segments {
				data := s
				if err := insertData(all, ld.Segments, key, &data); err != nil {
					return nil, err
				}
			}
		}
	}
	return all, nil
}

// Close is called automatically when the client is closed.
func (fs *fileDataSource) Close() (err error) {
	fs.closeOnce.Do(func() {
		if fs.closeReloaderCh != nil {
			close(fs.closeReloaderCh)
		}
	})
	return nil
}
