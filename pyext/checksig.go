package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"golang.org/x/sync/errgroup"
)

// CheckSignatures calculates sha1 signatures for files in rootDir and compare
// them with signature found at "sha1sum.txt" in the same directory. It'll
// return an error if one of the signatures don't match
func CheckSignatures(rootDir string) error {
	file, err := os.Open(path.Join(rootDir, "sha1sum.txt"))
	if err != nil {
		return err
	}
	defer file.Close()

	sigs, err := parseSigFile(file)
	if err != nil {
		return err
	}

	var g errgroup.Group
	for name, signature := range sigs {
		fileName := path.Join(rootDir, name)
		expected := signature
		g.Go(func() error {
			sig, err := fileSig(fileName)
			if err != nil {
				return err
			}
			if sig != expected {
				return fmt.Errorf("%q - mismatch", fileName)
			}
			return nil
		})
	}

	return g.Wait()
}

// fileSig returns the fileName sha1 digital signature of the specified file.
func fileSig(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// parseSigFile parses the signature file and returns a map of path->signature.
func parseSigFile(r io.Reader) (map[string]string, error) {
	sigs := make(map[string]string)
	scanner := bufio.NewScanner(r)
	lnum := 0

	for scanner.Scan() {
		lnum++

		// Line example: 6c6427da7893932731901035edbb9214 nasa-00.log
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			return nil, fmt.Errorf("%d: bad line: %q", lnum, scanner.Text())
		}
		sigs[fields[1]] = fields[0]
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return sigs, nil
}
