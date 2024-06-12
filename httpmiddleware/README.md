# What it is?

This package implements:

- creation of instrumented http.Server
- offer ready implementations for /status and /api-docs endpoints

# How to use it?

1. Create openapi.yaml file with desired endpoints
   
2. Ensure you add mandatory common /status and /api-docs endpoints
   
   ```yaml
    /status:
        get:
            summary: Docker probe for readiness
            responses:
            "200":
                description: successful operation

    /api-docs:
        get:
            summary: Swagger documentation endpoint
            responses:
            "200":
                description: successful operation
   ```

3. Code generate out of swagger
   
   ```shell
   make oapi-gen
   ```
   By end of it you should have "api" package generated.

4. Create "endpoints" package
   
5. Create serverimpl.go to host your swagger endpoints handing
     
   ```go
    type ServerImpl struct {
        log         logging.Logger
        swaggerJSON []byte
    }

    // New instantiates new ServerInterface implementation
    func New(log logging.Logger) (*ServerImpl, error) {
        var err error
        s := ServerImpl{
            log: log,
        }
        s.swaggerJSON, err = httpmiddleware.GetSwaggerJSON(api.GetSwagger)

        return &s, err
    }

    // GetApiDocs serves Swagger documentation endpoint (GET /api-docs)
    // nolint:golint,stylecheck
    func (s *ServerImpl) GetApiDocs(w http.ResponseWriter, r *http.Request) {
        httpmiddleware.GetApiDocs(w, r, s.swaggerJSON, s.log)
    }

    // GetStatus implements /status
    // nolint:golint,stylecheck
    func (s *ServerImpl) GetStatus(w http.ResponseWriter, r *http.Request) {
        httpmiddleware.GetStatus(w, r)
    }
   ```

6. In your main.go use it like following
   
   ```go
    // HTTPServer is abstracting needed http.Server methods
    type HTTPServer interface {
        ListenAndServe() error
        Shutdown(ctx context.Context) error
    }

    func newHTTPServer(config *EnvConfig, si api.ServerInterface) (HTTPServer, error) {
        httpMiddleware, err := httpmiddleware.New(api.Handler(si), api.GetSwagger)
        if err != nil {
            return nil, fmt.Errorf("failed creating http middleware: %s", err)
        }

        server := http.Server{
            Addr:    fmt.Sprintf("%s:%d", config.Addr, config.Port),
            Handler: httpMiddleware,
        }

        return &server, nil
    }
   ```
