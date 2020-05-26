NanoStatsD
----------
This is very simple statsd implementation that allows to debug server applications locally.
Now library uses https://github.com/zserge/metric/ as metrics rendering engine. I plan to
add support for dynamic rendering later, but not now.

Build
========
This project could be built by using go build command. If you modify HTML part you'll need
to run `make html`.

Usage
========
Typical usage is simple:
``` bash
nanostatsd -web localhost:8888 -stats localhost:8125
```
Specified parameters are defaults so `nanostatsd` will work directly.
