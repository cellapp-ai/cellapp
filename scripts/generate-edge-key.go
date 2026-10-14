package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func writeNew(path string, value []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(value)
	return err
}

func main() {
	if len(os.Args) != 3 || os.Args[1] == os.Args[2] {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/generate-edge-key.go PRIVATE_FILE PUBLIC_FILE")
		os.Exit(2)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err == nil {
		err = writeNew(os.Args[1], []byte(base64.RawURLEncoding.EncodeToString(private)), 0600)
	}
	if err == nil {
		err = writeNew(os.Args[2], []byte(base64.RawURLEncoding.EncodeToString(public)), 0644)
		if err != nil {
			_ = os.Remove(os.Args[1])
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "key generation failed:", err)
		os.Exit(1)
	}
	fmt.Println("edge signing key files created")
}
