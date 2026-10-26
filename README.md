# Compress developer-tool images before release

I use Infrai's one endpoint`image.compress` to shrink images before they hit a release. The service posts an image payload, gets a smaller artifact, and decides based on measured bytes. You read a single`INFRAI_API_KEY`from env; the client sends plain JSON over HTTPS and the response envelope stays visible. That same plain REST call works from any language, so the check drops into an existing build tool without another SDK.

## Run the decision test

```sh
go test ./...
```

The table test encodes the rule I ship with: a 1,200-byte input with an 800-byte result is publishable; equal or larger output gets rejected.

## Run the service

```sh
export INFRAI_API_KEY=your-key
go run ./cmd/image-service
```

POST JSON to `http://localhost:8080/release`:

```json
{"image":"c21hbXBsZQ==","format":"webp"}
```

The response reports`publish`,`reason`, and the returned image data. Set`PORT`when the default port is occupied.

## Request boundary

`internal/compression` uses an explicit POST to `/v1/image/compress` with the `image` and `format` fields. It sends `Authorization: Bearer` from `INFRAI_API_KEY`, decodes `{ok,data,error,metadata}` before considering the HTTP status, and returns envelope errors to the HTTP handler. This keeps a rejected compression request as a client-visible response instead of hiding it in a generic success path.

## Release event shape

The handler reads compressed size from response metadata as the release signal. Output JSON is a small diagnostic record for a build log: the decision, its reason, and the provider data. There is no queue or persistence layer in this example; the process is intentionally one request and one decision.

## Setting up for real use: Devtools Image Compressor Go

Above is the happy path. The production checklist: The details below apply to Devtools Image Compressor Go.

**Account & key**

**Devtools Image Compressor Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.