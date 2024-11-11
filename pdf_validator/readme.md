# PDF Validator

A Go program to scan a directory for PDF files, validate them, and optionally delete invalid or corrupted files. The program also generates a text file listing all valid and invalid PDF files.

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [Example](#example)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Directory Scanning**: Recursively or non-recursively scan a directory for PDF files.
- **PDF Validation**: Validate PDF files using the `pdfcpu` library.
- **File Deletion**: Optionally delete invalid or corrupted PDF files.
- **Output File**: Generate a text file listing all valid and invalid PDF files.

## Prerequisites

- Go (version 1.16 or higher)
- `pdfcpu` library

## Installation

1. **Clone the Repository**:
   ```sh
   git clone https://github.com/yourusername/pdf_validator.git
   cd pdf_validator
   ```

2. **Initialize a Go Module**:
   ```sh
   go mod init pdf_validator
   ```

3. **Add the Dependency**:
   ```sh
   go get github.com/pdfcpu/pdfcpu/pkg/api
   ```

## Usage

1. **Build the Program**:
   ```sh
   go build -o pdf_validator
   ```

2. **Run the Program**:
   ```sh
   ./pdf_validator
   ```

3. **Follow the Prompts**:
   - Enter the target directory path.
   - Choose whether to scan recursively.
   - Choose whether to delete invalid/corrupted files.

## Example

```sh
$ ./pdf_validator
Enter the target directory path: /path/to/pdfs
Do you want to scan recursively? (Y/N): Y
Do you want to delete invalid/corrupted files? (Y/N): N

Scanning and validating PDF files...
Validating: /path/to/pdfs/file1.pdf
  Valid
Validating: /path/to/pdfs/file2.pdf
  Invalid

Valid PDF files:
/path/to/pdfs/file1.pdf

Invalid PDF files:
/path/to/pdfs/file2.pdf

Suggested file 'validated_files.txt' with the list of valid and invalid PDF files.
```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature-branch`).
3. Make your changes.
4. Commit your changes (`git commit -am 'Add some feature'`).
5. Push to the branch (`git push origin feature-branch`).
6. Create a new Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---
