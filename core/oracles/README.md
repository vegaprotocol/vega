# Oracles

The Vega network runs on data. Market settlement, risk models, and other
features require a supplied price (or other data), which must come from
somewhere, often completely external to Vega. This necessitates the use of both
internal and external data sources for a variety of purposes.

These external data sources are called **oracles**.

Any features that want to consume oracle data can specify conditions on the data
they are expecting. This condition are defined inside an **oracle spec**.

