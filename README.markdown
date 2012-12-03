HTTP Proxy that submits a copy of each "interesting" page the user
visits to a logging webservice (eg,
https://github.com/thraxil/harken/).

Run with something like:

    go run stygian.go ~/config.json

See `sample-config.json` and the blacklist files for sample configuration.
