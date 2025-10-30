package utils

import (
    "io"
    "mime/multipart"
    "os"
    "path/filepath"
)

func EnsureDir(path string) error {
    return os.MkdirAll(path, 0o755)
}

func SaveFile(dstPath string, r io.Reader) error {
    if err := EnsureDir(filepath.Dir(dstPath)); err != nil { return err }
    f, err := os.Create(dstPath)
    if err != nil { return err }
    defer f.Close()
    _, err = io.Copy(f, r)
    return err
}

func UploadSingle(dstDir string, fileHeader *multipart.FileHeader) (string, error) {
    if err := EnsureDir(dstDir); err != nil { return "", err }
    src, err := fileHeader.Open()
    if err != nil { return "", err }
    defer src.Close()
    dst := filepath.Join(dstDir, filepath.Base(fileHeader.Filename))
    if err := SaveFile(dst, src); err != nil { return "", err }
    return dst, nil
}

func UploadMultiple(dstDir string, files []*multipart.FileHeader) ([]string, error) {
    paths := make([]string, 0, len(files))
    for _, fh := range files {
        p, err := UploadSingle(dstDir, fh)
        if err != nil { return nil, err }
        paths = append(paths, p)
    }
    return paths, nil
}

func DownloadTo(path string, src io.Reader) error {
    return SaveFile(path, src)
}
