HTTP Proxy that submits a copy of each "interesting" page the user
visits to a logging webservice (eg,
https://github.com/thraxil/harken/). "Interesting" means something
along the lines of "HTML or Text that doesn't look like ad iframes or
analytics includes and is likely to contain actual content worth
archiving or processing."

Run with something like:

    go run stygian.go ~/config.json

See `sample-config.json` and the blacklist files for sample configuration.
