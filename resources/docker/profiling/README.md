Profiling
=========

This docker compose sets a Benthos instance up with a custom config,
[Prometheus][prometheus], [Grafana][grafana] and [Jaeger][jaeger] in order to
observe its performance under different configurations.

# Set up

Edit the data within `sample_data.txt` to the type of input data you wish to
profile with.

Next, edit `benthos.yaml` in order to add processors and features that you wish
to profile against.

Run with `docker-compose up`.

Go to [http://localhost:3000](http://localhost:3000) in order to set up your own
dashboards.

Go to [http://localhost:16686](http://localhost:16686) in order to observe
opentracing events with Jaeger.

Use `go tool pprof http://localhost:4195/debug/pprof/profile` and similar
endpoints to get profiling data.

[prometheus]: https://prometheus.io/
[grafana]: https://grafana.com/
[jaeger]: https://www.jaegertracing.io/
