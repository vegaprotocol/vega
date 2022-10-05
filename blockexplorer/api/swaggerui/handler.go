package swaggerUI

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"code.vegaprotocol.io/vega/blockexplorer/api"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos"
	"github.com/getkin/kin-openapi/openapi2"
)

//go:embed assets
var swagfs embed.FS
var blockExplorerSpecPath = "generated/blockexplorer/swagger/blockexplorer/blockexplorer.swagger.json"

type SwaggerUI struct {
	api.SwaggerConfig
	restConfig api.RESTConfig
	log        *logging.Logger
	mux        http.Handler
}

func New(log *logging.Logger, config api.SwaggerConfig, restConfig api.RESTConfig) *SwaggerUI {
	log = log.Named("swagger-ui")

	r := &SwaggerUI{
		SwaggerConfig: config,
		restConfig:    restConfig,
		log:           log,
		mux:           api.NewNotStartedHandler("swagger-ui"),
	}

	return r
}

func (s *SwaggerUI) Name() string {
	return "swagger-ui"
}

func (s *SwaggerUI) Description() string {
	return "Interactive REST api documentation"
}

func (s *SwaggerUI) Start() error {
	static, _ := fs.Sub(swagfs, "assets")
	mux := http.NewServeMux()
	specHandler, err := s.specFile()
	if err != nil {
		return err
	}
	mux.HandleFunc("/swagger_spec", specHandler)
	mux.HandleFunc("/swagger-initializer.js", s.swaggerSetup())
	mux.Handle("/", http.FileServer(http.FS(static)))
	s.mux = mux
	return nil
}

func (s *SwaggerUI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.mux.ServeHTTP(w, req)
}

// specFile just returns the openapi2 spec file, but modified slightly to have the correct
// base path, so that the 'Try It Out' functionality works.
func (s *SwaggerUI) specFile() (http.HandlerFunc, error) {
	var spec openapi2.T

	originalSpec, err := fs.ReadFile(protos.Generated, blockExplorerSpecPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read embedded openapi json file '%s' (must be generated before build with 'make proto_docs')", blockExplorerSpecPath)
	}

	if err := json.Unmarshal(originalSpec, &spec); err != nil {
		return nil, fmt.Errorf("un-marshalling OpenAPI spec: %w", err)
	}

	spec.BasePath = s.restConfig.Endpoint

	newSpec, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling modified OpenAPI spec: %w", err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Write(newSpec)
	}, nil
}

// swaggerSetup returns the bit of javascript that sets up swagger, modified
// so that it points at our OpenAPI file endpoint.
func (s *SwaggerUI) swaggerSetup() http.HandlerFunc {
	template := `
window.onload = function () {
	window.ui = SwaggerUIBundle({
	  url: "%s",
	  name: "Block Explorer",
	  dom_id: '#swagger-ui',
	  deepLinking: true,
	  presets: [
		SwaggerUIBundle.presets.apis,
		SwaggerUIStandalonePreset
	  ],
	  plugins: [
		SwaggerUIBundle.plugins.DownloadUrl
	  ],
	  layout: "StandaloneLayout"
	});
  };
`
	js := []byte(fmt.Sprintf(template, s.Endpoint+"/swagger_spec"))
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Write(js)
	}
}
