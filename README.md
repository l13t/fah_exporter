# fah-exporter

`fah-exporter` is a tool designed to export Folding@Home statistics and metrics.

## Features

- Export Folding@Home statistics
- Supports multiple platforms (Linux, Windows, macOS)
- Easy to use

## Installation

### Using pre-built binaries

Download the latest release from the [Releases](https://github.com/your-username/fah-exporter/releases) page.

### Using Go

If you have Go installed, you can install `fah-exporter` using the following command:

```sh
go install github.com/your-username/fah-exporter@latest
```

### Using container image

If you have Docker installed, you can run `fah-exporter` using the following command:

```sh
docker run -p 9401:9401 ghcr.io/l13t/fah_exporter:latest -team-id <team-id> -user-name <username>
```

## Usage

```sh
❯ fah_exporter --help
Usage of fah_exporter:
  -fah-api-url string
        Base URL for Folding@Home API. (default "https://api.foldingathome.org")
  -listen-address string
        Address to listen on for HTTP requests. (default ":9401")
  -namespace string
        Namespace for the Prometheus metrics. (default "foldingathome")
  -team-id int
        Team ID to fetch stats for. (default -1)
  -user-name string
        User name to fetch stats for.
```

**Be aware**: teamID equal `-1` means you don't want to collect information about any team.

## Grafana dashboard

You can import [Grafana dashboard](fah_dashboard.json) into your Grafana instance.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## License

This project is licensed under the MIT License. See the [license](LICENSE) file for details.

## Contact

For any questions or feedback, please open an issue on the [GitHub repository](https://github.com/l13t/fah_exporter/issues/new).
