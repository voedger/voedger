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

// Copy directory recursively - even more complex
func copyDir(src, dst string) error {
    // 30+ lines of recursive directory walking,
    // error handling, permission copying...
    // Easy to get wrong, hard to maintain
}

srcInfo, err := srcFile.Stat()
if err != nil {
    return err
}
return os.Chmod(dstPath, srcInfo.Mode()) // Easy to forget
```
</details>

<details>
<summary>With filesu</summary>

```go
// Copy a single file
err := filesu.CopyFile("source.txt", "dest")

// Copy with options
err = filesu.CopyFile("source.txt", "dest",
    filesu.WithSkipExisting(),
    filesu.WithFileMode(0644))

// Copy entire directory
err = filesu.CopyDir("srcdir", "dstdir")
```
</details>

## Features

- **[CopyFile](impl.go#L37)** - Copy individual files with options
  - [File existence validation: impl.go#L99](impl.go#L99)
  - [Automatic directory creation: impl.go#L117](impl.go#L117)
  - [Permission preservation: impl.go#L142](impl.go#L142)
- **[CopyDir](impl.go#L45)** - Recursively copy directories
  - [Recursive directory traversal: impl.go#L69](impl.go#L69)
  - [Directory structure preservation: impl.go#L72](impl.go#L72)
- **[CopyFileFS](impl.go#L29)** - Copy from filesystem abstractions
  - [Filesystem abstraction support: types.go#L10](types.go#L10)
- **[CopyDirFS](impl.go#L20)** - Copy directories from filesystem abstractions
- **[Exists](impl.go#L151)** - Check file/directory existence safely
- **Copy options** - Flexible configuration
  - [WithSkipExisting: impl.go#L167](impl.go#L167)
  - [WithFileMode: impl.go#L161](impl.go#L161)
  - [WithNewName: impl.go#L173](impl.go#L173)
  - [WithFilterFilesWithRelativePaths: impl.go#L179](impl.go#L179)

## Use

See [example](example_test.go)
