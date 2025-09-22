# zipu

Simple directory compression and extraction utilities with automatic
conflict detection and cross-platform path handling.

## Problem

Standard Go archive operations require extensive boilerplate for
directory compression and lack built-in safeguards against common
pitfalls like overwriting existing files or including target files.

<details>
<summary>Without zipu</summary>

```go
// Manual zip creation with all the boilerplate
zipFile, err := os.Create("archive.zip")
if err != nil {
    return err
}
defer zipFile.Close()

zipWriter := zip.NewWriter(zipFile)
defer zipWriter.Close()

err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return err
    }
    
    // Skip root directory - easy to forget this check
    if path == sourceDir {
        return nil
    }
    
    // Manual relative path calculation - error prone
    relPath, err := filepath.Rel(sourceDir, path)
    if err != nil {
        return err
    }
    
    info, err := d.Info()
    if err != nil {
        return err
    }
    
    header, err := zip.FileInfoHeader(info)
    if err != nil {
        return err
    }
    
    // Cross-platform path handling - often forgotten
    header.Name = filepath.ToSlash(relPath)
    
    if info.IsDir() {
        header.Name += "/"
        _, err := zipWriter.CreateHeader(header)
        return err
    }
    
    header.Method = zip.Deflate
    writer, err := zipWriter.CreateHeader(header)
    if err != nil {
        return err
    }
    
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()
    
    _, err = io.Copy(writer, file)
    return err
})
```
</details>

<details>
<summary>With zipu</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/zipu"

// Compress directory
err := zipu.Zip("source/dir", "archive.zip")

// Extract archive
err = zipu.Unzip("archive.zip", "destination/dir")
```
</details>

## Features

- **[Zip](zip.go#L19)** - Compress directories with conflict detection
- **[Unzip](zip.go#L89)** - Extract archives with automatic directory creation

## Use

See [basic usage test](zip_test.go)