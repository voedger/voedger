# Filesu

Simplifies file and directory copying operations with flexible options
and filesystem abstraction support.

## Problem

Standard Go file operations require verbose boilerplate for common
copying tasks and lack built-in support for advanced options like
filtering, renaming, or handling existing files.

<details>
<summary>Without filesu</summary>

```go
// Copy a single file - lots of boilerplate
func copyFile(src, dst string) error {
    srcFile, err := os.Open(src)
    if err != nil {
        return err // Common mistake: not handling all error cases
    }
    defer srcFile.Close()

    // Boilerplate: check if destination directory exists
    dstDir := filepath.Dir(dst)
    if _, err := os.Stat(dstDir); os.IsNotExist(err) {
        if err := os.MkdirAll(dstDir, 0755); err != nil {
            return err
        }
    } else if err != nil {
        return err // Often forgotten: handle other stat errors
    }

    dstFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer dstFile.Close()

    // Copy content
    if _, err := io.Copy(dstFile, srcFile); err != nil {
        return err
    }

    // Sync to disk - often forgotten
    if err := dstFile.Sync(); err != nil {
        return err
    }

    // Copy permissions - manual work
    srcInfo, err := srcFile.Stat()
    if err != nil {
        return err
    }
    return os.Chmod(dst, srcInfo.Mode())
}
```
</details>

<details>
<summary>With filesu</summary>

```go
import "github.com/voedger/voedger/pkg/goutils/filesu"

// Copy a single file
err := filesu.CopyFile("source.txt", "dest")

// Simple directory copy
err := filesu.CopyDir("/path/to/source", "/path/to/destination")

// Copy with options - skip existing files and set custom permissions
err = filesu.CopyDir(src, dst, 
    filesu.WithSkipExisting(),
    filesu.WithFileMode(0755))

// Copy from embedded filesystem
err = filesu.CopyDirFS(embedFS, "templates", "/output/dir")
```
</details>

## Features

- **[CopyDir](impl.go#L49)** - Recursively copy directories on disk
- **[CopyFile](impl.go#L39)** - Copy single files with auto directory creation
- **Embedded filesystem support** - Copy from fs.FS implementations
  - [CopyDirFS: embedded directory copying](impl.go#L21)
  - [CopyFileFS: embedded file copying](impl.go#L29)
  - [IReadFS: filesystem abstraction interface](types.go#L10)
- **Configuration options** - Flexible copy behavior control
  - [WithFileMode: custom file permissions](impl.go#L169)
  - [WithSkipExisting: avoid overwrite conflicts](impl.go#L175)
  - [WithNewName: rename during copy](impl.go#L181)
  - [WithFilterFilesWithRelativePaths: selective copying](impl.go#L187)
- **[Exists](impl.go#L157)** - Safe file/directory existence checking

## Use

See [example](example_test.go)
