# PubSub gRPC Service

Simple in‑memory **publisher/ subscriber** bus with a thin gRPC.

* **Logging**: logrus (JSON or text – configurable)
* **Config**: single `config.yaml` (overridden by ENV)
* **Graceful shutdown**

---

## Directory layout

```
├── api/                 # *.proto and generated stubs
├── cmd/pubsubd/         # main()
├── internal/
│   ├── app/             
│   ├── grpcserver/      # Run() helper with graceful shutdown
│   ├── usecase/         # gRPC handlers (Publish, Subscribe)
│   └── config/          # YAML+ENV loader
├── pkg/subpub/          # pure Go FIFO pub‑sub bus
└── config.yaml          # local config
```

---

## Configuration (`config.yaml`)

```yaml
grpc:
  host: "0.0.0.0"   # listen interface
  port: 9090        # listen port
log:
  level: "info"      # trace | debug | info | warn | error | fatal
```

Example:

```bash
LOG_LEVEL=debug GRPC_PORT=8080 go run ./cmd/pubsubd
```

---

## Running locally

```bash
go run ./cmd/pubsubd            
```

```bash
# Subscribe (stream stays open)
grpcurl -plaintext \
  -import-path ./api \
  -proto pubsub/v1/pubsub.proto \
  -d '{"key":"orders"}' \
  localhost:8080 api.pubsub.v1.PubSub/Subscribe

# Publish an event
grpcurl -plaintext \
  -import-path ./api \
  -proto pubsub/v1/pubsub.proto \
  -d '{"key":"orders","data":"order-42"}' \
  localhost:8080 api.pubsub.v1.PubSub/Publish
```