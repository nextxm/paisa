# Paisa

[![Matrix](https://img.shields.io/matrix/paisa%3Amatrix.org?logo=matrix)](https://matrix.to/#/#paisa:matrix.org)

**Paisa** is a Personal finance manager. It builds on
top of the [ledger](https://www.ledger-cli.org/) double entry accounting tool. Checkout
[documentation](https://nextxm.github.io/paisa/) or view the [Roadmap](./docs/roadmap.md) to see where we're headed.



## Scheduled Price Sync

Paisa's backend (`paisa serve`) does not have a built-in background daemon for scheduling tasks. Instead, it exposes the `update` command which is specifically designed for synchronizing data. To schedule periodic updates of commodity prices, use your operating system's task scheduler to run:

```bash
paisa update --commodity
```

* **Windows**: Use **Task Scheduler** to create a basic task that runs daily, executing `paisa.exe update --commodity`.
* **Linux / macOS**: Add a cron job (`crontab -e`) to run it daily, for example: `0 18 * * * /path/to/paisa update --commodity`
* **Docker**: Use host cron to execute `docker exec <container_name> paisa update -c`, or add a lightweight cron sidecar container sharing the database volume.

## Status

I use it to track my personal finance. Most of my personal use cases
are covered. Feel free to open an issue if you found a bug or start a
discussion if you have a feature request. If you have any question,
you can ask on [Matrix chat](https://matrix.to/#/#paisa:matrix.org).

## License

This software is licensed under [the AGPL 3 or later license](./COPYING).
This repository is a fork of [ananthakumaran/paisa](https://github.com/ananthakumaran/paisa),
and active updates for this fork are available at
[nextxm/paisa](https://github.com/nextxm/paisa).
