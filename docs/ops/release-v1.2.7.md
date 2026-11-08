## Changes

- Pin Geminio v1.3.0, including negotiated flow control, reset/cancellation recovery and reset-notification admission backpressure.
- Recover Edge ownership after migration without restarting service callers. Only the affected Edge route is refreshed; other Frontier connections remain usable. No automatic business-payload replay.
- Restore missing Edge registrations on heartbeat and recover service registrations with independent leases in Frontlas.
- Omit opaque metadata and credential-like SDK diagnostic values from logs.
- Add monitored concurrent-stream benchmarks and documented recovery/soak results.

## Compatibility and requirements

- Minimum Go version is 1.25 due to the Geminio dependency. Public wire changes are negotiated; legacy peers keep the legacy behavior.
- Frontier and Frontlas share release version 1.2.7.
- Linux archive binaries use CGO_ENABLED=0. Deployments requiring a CGO-only feature must build separately with that feature enabled.
- Standalone images run as UID/GID 65532. Configuration defaults to `/usr/conf/server.yaml`; mount your own configuration and pass `--config` as needed.

## Validation and limitations

The focused recovery/race gate (`make verify-pr`) covers stream reset isolation, service/Edge registration recovery, route recovery, multi-Edge connections and vet. The repository still tracks legacy full-suite fixture failures separately; this is not a claim that every historical test passes.

Geminio reset-pressure validation completed 7,300,895 operations at 512/1024 concurrency with zero failures/reconnects. The corresponding local Trustio deployment passed 19,199 real desktop TUN HTTP/TCP requests and five domain/CIDR TCP/UDP/ICMP probe rounds. These finite tests are not production capacity guarantees.

## Artifacts

Linux amd64/arm64 archives include the executable and example YAML. Verify downloads using `SHA256SUMS`. Published image addresses and immutable digests are recorded in `images.json`.
