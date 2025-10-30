package utils

import (
    "encoding/json"
    "os"
)

func ToJSON(v any) (string, error) {
    b, err := json.Marshal(v)
    if err != nil { return "", err }
    return string(b), nil
}

func FromJSON[T any](s string, out *T) error {
    return json.Unmarshal([]byte(s), out)
}

func WriteJSONFile(path string, v any) error {
    b, err := json.MarshalIndent(v, "", "  ")
    if err != nil { return err }
    return os.WriteFile(path, b, 0o644)
}
