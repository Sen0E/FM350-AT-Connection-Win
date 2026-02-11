# FM350 AT Connection Win

A Go-based serial communication program for Windows platform, designed to interact with FM350 devices via AT commands.

## Project Overview

`fm350_at_connection_win` is a serial communication tool specifically designed for Windows platform to establish stable connections with FM350 modules and perform AT command interactions. 

## Main Features

- Open and configure serial ports
- Send AT commands to FM350 modules
- Receive and output module response data
- Handle Chinese character encoding
## Tech Stack

- **Programming Language**: Go
- **Serial Communication Library**: [go.bug.st/serial](https://github.com/bugst/go-serial)
- **Text Encoding Handling**: [golang.org/x/text](https://pkg.go.dev/golang.org/x/text)

## Usage Instructions

1. Ensure the FM350 device is properly connected to the serial port of the Windows system
2. When running the program, you may need to specify the correct COM port number
3. The program will automatically initialize the serial connection and allow sending AT commands
4. Received response data will be properly decoded and displayed in the terminal

## Notes

- This program is specifically designed for Windows platform and may not work properly on other operating systems
- Ensure that the user running the program has permission to access the serial port device
- If you encounter Chinese character display issues, ensure that the terminal supports the corresponding character encoding

## License

MIT [LICENSE](LICENSE)