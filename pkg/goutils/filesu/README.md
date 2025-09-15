# filesu

File system utilities for Go applications.

## Problem

Copying files and directories in Go requires verbose boilerplate code with manual error handling, directory creation, and permission management.

<details>
<summary>Before filesu</summary>

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

// Copy directory recursively - even more complex
func copyDir(src, dst string) error {
    // 30+ lines of recursive directory walking,
    // error handling, permission copying...
    // Easy to get wrong, hard to maintain
}
```
</details>

<details>
<summary>Now filesu</summary>

```go
// Copy a single file - one line
err := filesu.CopyFile("source.txt", "/destination/dir")

// Copy entire directory - one line
err := filesu.CopyDir("/source/dir", "/destination/dir")

// Copy with options - still simple
err := filesu.CopyFile("source.txt", "/dest",
    filesu.WithSkipExisting(),
    filesu.WithFileMode(0644))
```
</details>

## Features

- **[CopyFile](impl.go#L37)** - Copy single files with automatic directory creation
- **[CopyDir](impl.go#L45)** - Recursively copy entire directories
- **CopyFS operations** - Copy from embedded or custom filesystems
  - [CopyFileFS: copy single file from fs.FS](impl.go#L29)
  - [CopyDirFS: copy directory from custom filesystem](impl.go#L20)
  - [IReadFS interface: filesystem abstraction](types.go#L10)
- **Configuration options** - Flexible copying behavior
  - [WithFileMode: set custom file permissions](impl.go#L161)
  - [WithSkipExisting: avoid overwriting files](impl.go#L167)
  - [WithNewName: rename during copy](impl.go#L173)
  - [WithFilterFilesWithRelativePaths: selective copying](impl.go#L179)
- **[Exists](impl.go#L151)** - Safe file existence checking

## Use

See [example](example_test.go)
