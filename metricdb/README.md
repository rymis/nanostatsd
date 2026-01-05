Metrics database
================

This package contains implementation of a database for metrics. It is not the timeseries database, it has been built to be very memory efficient, but not 100% precise.

To obtain this database does not store all the values. Instead it stores precomputed histograms/statistics. These parts can be merged later and used for storing metrics.

Architecture
------------

Database contains three layers:

1. Real-time layer (RTL) is used for the last 15 seconds of data. This layer contains raw values.
2. Middle-range layer (MTL) contains pre-computed histograms for 15-second intervals.
3. Long-time layer (LTL) merges 15-second intervals to 15 minutes intervals and stores them in LevelDB.

**RTL** - is stored in memory only. They are not restored in case of restart.

**MTL** - is written to files quant-{Q}.json. When full 15-minutes interval is collected it is merged into the main database.

**LTL** - is stored in LevelDB.

