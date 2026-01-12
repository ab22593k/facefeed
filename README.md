# fbuplish

A robust CLI tool for publishing content to Facebook Pages, supporting text, links, and multiple images (including SVGs).

## Features

- **Text & Link Posts**: Quick sharing of updates and URLs.
- **Multiple Image Uploads**: Upload one or many photos in a single command.
- **Local & Remote Support**: Accepts local file paths or public image URLs.
- **SVG Support**: Automatically converts SVG files to PNG (requires ImageMagick).
- **Progress Tracking**: Real-time upload progress bars for local files.
- **Rollback**: Easily delete the latest post or a specific post by ID.
- **Dry-Run Mode**: Validate your inputs and file sizes before publishing.
- **Environment Support**: Load credentials and default content from a `.env` file.

## Installation

1. Clone the repository.
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. (Optional) Install ImageMagick for SVG support:
   ```bash
   sudo apt-get install imagemagick  # Linux
   brew install imagemagick          # macOS
   ```

## Configuration

Create a `.env` file in the project root:

```env
FB_PAGE_ID=your_page_id
FB_ACCESS_TOKEN=your_page_token

# Optional defaults
FB_MESSAGE="Default caption"
FB_IMAGES="photo1.jpg,photo2.jpg"
```

## Usage

### Publishing

```bash
# Basic text post
go run main.go -message "Hello World"

# Multiple images (local or URL)
go run main.go -message "Album" -image ./img1.jpg -image https://example.com/img2.png

# SVG upload (auto-converted to PNG)
go run main.go -image logo.svg -message "Our new logo"
```

### Management

```bash
# Rollback latest post
go run main.go -rollback

# Delete a specific post
go run main.go -rollback <post_id>

# Validate without publishing
go run main.go -dry-run -image large_photo.jpg -message "Test"
```
