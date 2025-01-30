# DNS Speed Test

A powerful and user-friendly DNS server speed testing tool with a modern graphical interface. Test and compare the performance of various DNS providers to find the fastest and most reliable DNS servers for your network.

## Features

- 🚀 Test multiple popular DNS providers simultaneously
- 📊 Beautiful graphical user interface
- 📈 Real-time results display
- 📑 Test history tracking
- 💾 Export results to CSV
- 🌐 Support for both IPv4 and IPv6
- 🔄 Configurable test parameters
- 🔌 TCP/UDP protocol support

## Pre-built Binaries

You can find pre-built binaries in the releases section of this repository.

## Building from Source

### Prerequisites

- Go 1.16 or later
- Git

### Installation

1. Clone the repository:
```bash
git clone [your-repository-url]
cd [repository-name]
```

2. Install dependencies:
```bash
cd GO
go mod download
```

3. Build the application:
```bash
go build -ldflags -H=windowsgui dns_speed_test_gui.go
```

## Usage

1. Launch the application by running the executable
2. Select the DNS providers you want to test
3. Configure test parameters (optional):
   - Number of tests per domain
   - Timeout duration
   - TCP/UDP protocol
   - IPv4/IPv6 preference
   - Parallel/Sequential testing
4. Click "Start Test" to begin the speed test
5. View results in real-time
6. Export results to CSV if desired

## Configuration Options

- **Tests Per Domain**: Number of queries to run for each test domain
- **Timeout**: Maximum wait time for DNS responses
- **Protocol**: Choose between UDP (default) or TCP
- **IP Version**: Test using IPv4, IPv6, or both
- **Test Mode**: Run tests in parallel or sequentially

## Contributing

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and coding conventions
- Add comments for non-obvious code sections
- Update documentation for new features
- Add tests for new functionality
- Ensure the GUI remains responsive during operations

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
This means you can:
- Use the software for any purpose
- Change the software to suit your needs
- Share the software with your friends and neighbors
- Share the changes you make

You cannot:
- Use this software for commercial purposes without sharing your source code
- Distribute this software without providing the source code
- Use this software as part of a proprietary system

## Acknowledgments

- Thanks to all DNS providers for their public DNS services
- Built with [Gio UI](https://gioui.org/) for the graphical interface